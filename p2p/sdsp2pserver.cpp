#include "sdsp2pserver.h"

#include "p2p_common.h"
#include "common/loguru.hpp"

#include <arpa/inet.h>
#include <sys/socket.h>
#include <poll.h>

#include <cstdint>
#include <cstring>
#include <unistd.h>
#include <unordered_map>

#define MAX_CLIENT_COUNT  100
#define MAX_POLL_FD_COUNT (MAX_CLIENT_COUNT+1)

int SdsP2PServer::ping(SdsBytesBuf &args, SdsBytesBuf &reply)
{
    PingArgs pingArgs;
    pingArgs.read(args);

    this->localNode->ping(pingArgs.id, pingArgs.address.c_str());

    return ERR_NULL;
}

int SdsP2PServer::findNode(SdsBytesBuf &args, SdsBytesBuf &reply)
{
    FindNodeArgs findArgs;
    findArgs.read(args);

    FindNodeReply findReply;
    this->localNode->findNode(findReply.nearest, findArgs.targetId);

    findReply.write(reply);
    return ERR_NULL;
}

int SdsP2PServer::storeResult(SdsBytesBuf &args, SdsBytesBuf &reply)
{
    StoreResultArgs storeArgs;
    storeArgs.read(args);

    this->localNode->storeResult(storeArgs.se);

    return ERR_NULL;
}

int SdsP2PServer::findResults(SdsBytesBuf &args, SdsBytesBuf &reply)
{
    FindResultsArgs findArgs;
    findArgs.read(args);

    FindResultsReply findReply;
    this->localNode->findResults(findReply.nearest, findReply.results, findArgs.query.c_str());

    findReply.write(reply);

    return ERR_NULL;
}

void SdsP2PServer::handleRequest(int client_fd)
{
    uint64_t recv_sz;
    std::string errstr = "";
    MessageRequestHeader req;
    MessageResponseHeader reply;

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

    recv_sz = le64toh(req.datasize);
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
    reply.id = req.id;

    switch (req.funcode) {
    case FUNC_PING:
        reply.errcode = this->ping(argsBuf, replyBuf);
        break;
    case FUNC_FIND_NODE:
        reply.errcode = this->findNode(argsBuf, replyBuf);
        break;
    case FUNC_STORE_RESULT:
        reply.errcode = this->storeResult(argsBuf, replyBuf);
        break;
    case FUNC_FIND_RESULTS:
        reply.errcode = this->findResults(argsBuf, replyBuf);
        break;
    default:
        reply.errcode = ERR_NOFUNCT;
        break;
    }

    if (reply.errcode == ERR_NULL) {
        reply.datasize = htole64(replyBuf.size());
    }

rpc_fail:
    send(client_fd, &reply, sizeof(reply), 0);

    if (replyBuf.size() > 0 && reply.errcode == ERR_NULL) {
        send(client_fd, replyBuf.bufPtr(), replyBuf.size(), 0);
    }
    // do not close conn the client will close them if needed

    LOG_F(1, "p2p server error %d: %s", reply.errcode, p2p_strerror(reply.errcode));
}

SdsP2PServer::SdsP2PServer(LocalNode *node)
    : running(true), localNode(node)
{}

SdsP2PServer::~SdsP2PServer()
{
    close(this->server_fd);
}

int SdsP2PServer::startListening(const char *addrstr, int port)
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

    int clients_count = 0;
    pollfd wait_fds[MAX_POLL_FD_COUNT];
    memset(&wait_fds, 0, sizeof(wait_fds));

    wait_fds[0].fd = this->server_fd;
    wait_fds[0].events = POLLIN | POLLPRI;
    while (this->running) {
        if (poll(wait_fds, MAX_POLL_FD_COUNT, 300) > 0) {
            int i;
            if ((wait_fds[0].revents & POLLIN)) {
                if (clients_count >= MAX_CLIENT_COUNT) {
                    continue;
                }

                client_fd = accept(this->server_fd, (struct sockaddr*) &address, &addrlen);
                LOG_F(1, "new p2p connection from %s", inet_ntoa(address.sin_addr));
                for (i = 1; i <= MAX_POLL_FD_COUNT; i++) {
                    if (wait_fds[i].fd == 0) {
                        clients_count++;
                        wait_fds[i].fd = client_fd;
                        wait_fds[i].events = POLLIN | POLLPRI;
                        break;
                    }
                }
            }

            for (i = 1; i <= MAX_POLL_FD_COUNT; i++) {
                client_fd = wait_fds[i].fd;
                short int revents = wait_fds[i].revents;
                if (client_fd > 0 && revents > 0) {
                    if ((revents & POLLHUP) || (revents & POLLERR)) {
                        close(wait_fds[i].fd);
                    } else if (revents & POLLIN) {
                        this->handleRequest(client_fd);
                    }

                    wait_fds[i].fd = 0;
                    wait_fds[i].revents = 0;
                    clients_count--;
                }
            }
        }
    }

    return 0;
}

void SdsP2PServer::shutdown()
{
    this->running = false;
}
