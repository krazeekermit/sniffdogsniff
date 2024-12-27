#ifndef SDSRPCSERVER_H
#define SDSRPCSERVER_H

#include "localnode.h"

#include "msgpack11/msgpack11.h"

#include <pthread.h>

#include <deque>

class SdsRpcServer
{
public:
    SdsRpcServer(LocalNode *node);
    ~SdsRpcServer();

    int startListening(const char *addrstr, int port);
    void shutdown();

private:
    static void *handleRequest(void *srvp);
    static int ping(SdsRpcServer *srv, msgpack11::MsgPack &args, msgpack11::MsgPack &reply);
    static int findNode(SdsRpcServer *srv, msgpack11::MsgPack &args, msgpack11::MsgPack &reply);
    static int storeResult(SdsRpcServer *srv, msgpack11::MsgPack &args, msgpack11::MsgPack &reply);
    static int findResults(SdsRpcServer *srv, msgpack11::MsgPack &args, msgpack11::MsgPack &reply);

    int running;

    LocalNode *localNode;
    std::deque<int> clientsQueue;
    pthread_mutex_t mutex;
    pthread_cond_t cond;
    pthread_t *threadPool;
    int server_fd;
};

#endif // SDSRPCSERVER_H
