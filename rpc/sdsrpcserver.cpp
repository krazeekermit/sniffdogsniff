#include "sdsrpcserver.h"

#include "rpc_common.h"
#include "common/logging.h"

#include <arpa/inet.h>
#include <sys/socket.h>

#include <cstdint>
#include <cstring>
#include <unistd.h>
#include <unordered_map>

#define THREAD_POOL_SZ    2

int threadNo = 0;

int SdsRpcServer::ping(SdsRpcServer *srv, SdsBytesBuf &args, SdsBytesBuf &reply)
{
    PingArgs pingArgs;
    pingArgs.read(args);

    srv->localNode->ping(pingArgs.callerId, pingArgs.callerAddress.c_str());

    return ERR_NULL;
}

int SdsRpcServer::findNode(SdsRpcServer *srv, SdsBytesBuf &args, SdsBytesBuf &reply)
{
    FindNodeArgs findArgs;
    findArgs.read(args);
    srv->localNode->nodeConnected(findArgs.callerId, findArgs.callerAddress);

    FindNodeReply findReply;
    srv->localNode->findNode(findReply.nearest, findArgs.targetId);

    findReply.write(reply);
    return ERR_NULL;
}

int SdsRpcServer::storeResult(SdsRpcServer *srv, SdsBytesBuf &args, SdsBytesBuf &reply)
{
    StoreResultArgs storeArgs;
    storeArgs.read(args);

    srv->localNode->nodeConnected(storeArgs.callerId, storeArgs.callerAddress);
    srv->localNode->storeResult(storeArgs.se);

    return ERR_NULL;
}

int SdsRpcServer::findResults(SdsRpcServer *srv, SdsBytesBuf &args, SdsBytesBuf &reply)
{
    FindResultsArgs findArgs;
    findArgs.read(args);
    srv->localNode->nodeConnected(findArgs.callerId, findArgs.callerAddress);

    FindResultsReply findReply;
    srv->localNode->findResults(findReply.results, findArgs.query.c_str());

    findReply.write(reply);

    return ERR_NULL;
}

struct RequestHandler {
    uint8_t funcode;
    int (*funptr) (SdsRpcServer *srv, SdsBytesBuf &args, SdsBytesBuf &reply);
};

void *SdsRpcServer::handleRequest(void *srvp)
{
    static const RequestHandler handlers[4] = {
        {.funcode=FUNC_PING,            .funptr=&SdsRpcServer::ping},
        {.funcode=FUNC_FIND_NODE,       .funptr=&SdsRpcServer::findNode},
        {.funcode=FUNC_STORE_RESULT,    .funptr=&SdsRpcServer::storeResult},
        {.funcode=FUNC_FIND_RESULTS,    .funptr=&SdsRpcServer::findResults},
    };
    threadNo++;
    int me = threadNo;
    int i, client_fd;
    uint64_t recv_sz;
    std::string errstr = "";
    RpcRequestHeader req;
    RpcResponseHeader reply;
    SdsRpcServer *srv = static_cast<SdsRpcServer*>(srvp);
    while (srv->running) {
        pthread_mutex_lock(&srv->mutex);
        while (srv->clientsQueue.empty()) {
            if (!srv->running) {
                logdebug << "server thread down..." << me;
                pthread_mutex_unlock(&srv->mutex);
                return nullptr;
            }
            pthread_cond_wait(&srv->cond, &srv->mutex);
        }
        client_fd = srv->clientsQueue.front();
        srv->clientsQueue.pop_front();
        pthread_mutex_unlock(&srv->mutex);

        memset(&req, 0, sizeof(req));
        memset(&reply, 0, sizeof(reply));

        SdsBytesBuf argsBuf;
        SdsBytesBuf replyBuf;

        recv_sz = sizeof(req);
        if (recv(client_fd, &req, recv_sz, 0) != recv_sz) {
            //logerror or send error
            reply.errcode = ERR_RECV_REQUEST;
            goto rpc_fail;
        }

        recv_sz = req.datasize;
        if (recv_sz > 0) {
            argsBuf.allocate(recv_sz);
            if (recv(client_fd, argsBuf.bufPtr(), recv_sz, 0) != recv_sz) {
                //logerror or send error
                reply.errcode = ERR_RECV_REQUEST;
                goto rpc_fail;
            }
        }

        reply.errcode = ERR_NOFUNCT;
        reply.funcode = req.funcode;
        memcpy(&reply.id, &req.id, ID_SIZE);

        for (i = 0; i < 4; i++) {
            RequestHandler handler = handlers[i];
            if (handler.funcode == req.funcode) {
                reply.errcode = handler.funptr(srv, argsBuf, replyBuf);
                break;
            }
        }

rpc_fail:
        send(client_fd, &reply, sizeof(reply), 0);

        if (replyBuf.size() > 0) {
            send(client_fd, replyBuf.bufPtr(), replyBuf.size(), 0);
        }
        close(client_fd);

        logdebug << "rpcServer error " << reply.errcode << ": " << rpc_strerror(reply.errcode);
    }

    return nullptr;
}

SdsRpcServer::SdsRpcServer(LocalNode *node)
    : running(true), threadPool(nullptr), localNode(node)
{
    pthread_mutex_init(&this->mutex, nullptr);
    pthread_cond_init(&this->cond, nullptr);
}

SdsRpcServer::~SdsRpcServer()
{
    this->shutdown();
    pthread_mutex_destroy(&this->mutex);
    pthread_cond_destroy(&this->cond);
}

int SdsRpcServer::startListening(const char *addrstr, int port)
{
    int i, fd, client_fd;
    ssize_t valread;
    struct sockaddr_in address;
    int opt = 1;
    socklen_t addrlen = sizeof(address);

    fd = socket(AF_INET, SOCK_STREAM, 0);
    if (fd < 0)
        return -1;

    if (setsockopt(fd, SOL_SOCKET, SO_REUSEADDR | SO_REUSEPORT, &opt, sizeof(opt))) {
        return -1;
    }

    if (inet_pton(AF_INET, addrstr, &address.sin_addr) <= 0) {
        return -1;
    }

    address.sin_family = AF_INET;
    address.sin_port = htons(port);

    if (bind(fd, (struct sockaddr*)&address, sizeof(address)) < 0) {
        return -2;
    }
    if (listen(fd, 3) < 0) {
        return -2;
    }

    this->server_fd = fd;
    if (!this->threadPool) {
        this->threadPool = new pthread_t[THREAD_POOL_SZ];
        for (i = 0; i < THREAD_POOL_SZ; i++) {
            pthread_create(&this->threadPool[i], nullptr, &SdsRpcServer::handleRequest, this);
        }
    }

    while ((client_fd = accept(fd, (struct sockaddr*)&address, &addrlen)) > -1) {
        pthread_mutex_lock(&this->mutex);
        this->clientsQueue.push_back(client_fd);
        pthread_cond_signal(&this->cond);
        pthread_mutex_unlock(&this->mutex);
    }

    return 0;
}

void SdsRpcServer::shutdown()
{
    if (this->threadPool) {
        pthread_mutex_lock(&this->mutex);
        this->running = false;
        pthread_cond_broadcast(&this->cond);
        pthread_mutex_unlock(&this->mutex);

        int i;
        void *dummy = nullptr;
        for (i = 0; i < THREAD_POOL_SZ; i++) {
            pthread_join(this->threadPool[i], &dummy);
        }

        close(this->server_fd);

        delete[] this->threadPool;
        this->threadPool = nullptr;
    }
}
