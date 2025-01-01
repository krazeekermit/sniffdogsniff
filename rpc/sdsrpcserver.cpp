#include "sdsrpcserver.h"

#include "rpc_common.h"
#include "logging.hpp"

#include <arpa/inet.h>
#include <sys/socket.h>

#include <cstdint>
#include <cstring>
#include <unistd.h>
#include <unordered_map>

#define THREAD_POOL_SZ    4

int SdsRpcServer::ping(SdsRpcServer *srv, msgpack11::MsgPack &args, msgpack11::MsgPack &reply)
{
    PingArgs pingArgs;
    if (!pingArgs.unpack(args))
        return ERR_SERIALIZE;

    srv->localNode->ping(pingArgs.id, pingArgs.address.c_str());

    return ERR_NULL;
}

int SdsRpcServer::findNode(SdsRpcServer *srv, msgpack11::MsgPack &args, msgpack11::MsgPack &reply)
{
    FindNodeArgs findArgs;
    if (!findArgs.unpack(args))
        return ERR_SERIALIZE;

    FindNodeReply findReply;
    srv->localNode->findNode(findReply.nearest, findArgs.id);

    findReply.pack(reply);
    return ERR_NULL;
}

int SdsRpcServer::storeResult(SdsRpcServer *srv, msgpack11::MsgPack &args, msgpack11::MsgPack &reply)
{
    StoreResultArgs storeArgs;
    if (!storeArgs.unpack(args))
        return ERR_SERIALIZE;

    srv->localNode->storeResult(storeArgs.se);

    return ERR_NULL;
}

int SdsRpcServer::findResults(SdsRpcServer *srv, msgpack11::MsgPack &args, msgpack11::MsgPack &reply)
{
    FindResultsArgs findArgs;
    if (!findArgs.unpack(args))
        return ERR_SERIALIZE;

    FindResultsReply findReply;

    srv->localNode->findResults(findReply.results, findArgs.query.c_str());

    findReply.pack(reply);

    return ERR_NULL;
}

struct RequestHandler {
    uint8_t funcode;
    int (*funptr) (SdsRpcServer *srv, msgpack11::MsgPack &args, msgpack11::MsgPack &reply);
};

void *SdsRpcServer::handleRequest(void *srvp)
{
    static const RequestHandler handlers[4] = {
        {.funcode=FUNC_PING,            .funptr=&SdsRpcServer::ping},
        {.funcode=FUNC_FIND_NODE,       .funptr=&SdsRpcServer::findNode},
        {.funcode=FUNC_STORE_RESULT,    .funptr=&SdsRpcServer::storeResult},
        {.funcode=FUNC_FIND_RESULTS,    .funptr=&SdsRpcServer::findResults},
    };

    int i, client_fd;
    char *argsbuf;
    uint64_t recv_sz;
    std::string errstr = "";
    msgpack11::MsgPack args;
    msgpack11::MsgPack resp;
    RpcRequestHeader req;
    RpcResponseHeader reply;
    SdsRpcServer *srv = static_cast<SdsRpcServer*>(srvp);
    while (srv->running) {
        pthread_mutex_lock(&srv->mutex);
        while (srv->clientsQueue.empty()) {
            pthread_cond_wait(&srv->cond, &srv->mutex);
        }
        client_fd = srv->clientsQueue.front();
        srv->clientsQueue.pop_front();
        pthread_mutex_unlock(&srv->mutex);

        memset(&req, 0, sizeof(req));
        memset(&reply, 0, sizeof(reply));

        argsbuf = nullptr;

        recv_sz = sizeof(req);
        if (recv(client_fd, &req, recv_sz, 0) != recv_sz) {
            //logerror or send error
            reply.errcode = ERR_RECV_REQUEST;
            goto rpc_fail;
        }

        recv_sz = req.datasize;
        argsbuf = new char[recv_sz];
        if (recv(client_fd, argsbuf, recv_sz, 0) != recv_sz) {
            //logerror or send error
            reply.errcode = ERR_RECV_REQUEST;
            goto rpc_fail;
        }

        args = msgpack11::MsgPack::parse(argsbuf, recv_sz, errstr);

        reply.errcode = ERR_NOFUNCT;
        reply.funcode = req.funcode;
        memcpy(&reply.id, &req.id, ID_SIZE);

        for (i = 0; i < 4; i++) {
            RequestHandler handler = handlers[i];
            if (handler.funcode == req.funcode) {
                reply.errcode = handler.funptr(srv, args, resp);
                break;
            }
        }

rpc_fail:
        std::string respbuf = resp.dump();
        reply.datasize = respbuf.length();

        send(client_fd, &reply, sizeof(reply), 0);

        send(client_fd, respbuf.c_str(), respbuf.length(), 0);
        close(client_fd);

        delete[] argsbuf;

        logdebug << "rpcServer error " << reply.errcode << ": " << rpc_strerror(reply.errcode);
    }

    return nullptr;
}

SdsRpcServer::SdsRpcServer(LocalNode *node)
    : running(1), threadPool(nullptr), localNode(node)
{
    pthread_mutex_init(&this->mutex, nullptr);
    pthread_cond_init(&this->cond, nullptr);
}

int SdsRpcServer::startListening(const char *addrstr, int port)
{
    int i, fd, client_fd;
    ssize_t valread;
    struct sockaddr_in address;
    int opt = 1;
    socklen_t addrlen = sizeof(address);

    if (!this->threadPool) {
        this->threadPool = new pthread_t[THREAD_POOL_SZ];
        for (i = 0; i < THREAD_POOL_SZ; i++) {
            pthread_create(&this->threadPool[i], nullptr, &SdsRpcServer::handleRequest, this);
        }
    }

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

    while ((client_fd = accept(fd, (struct sockaddr*)&address, &addrlen)) > -1) {
        this->clientsQueue.push_back(client_fd);
        pthread_cond_signal(&this->cond);
    }

    return 0;
}

void SdsRpcServer::shutdown()
{
    int i;
    void *punused = nullptr;
    this->running = 0;
    for (i = 0; i < THREAD_POOL_SZ; i++) {
        pthread_join(this->threadPool[i], &punused);
    }
    delete[] this->threadPool;
    this->threadPool = nullptr;
}
