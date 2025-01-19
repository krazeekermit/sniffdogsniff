#ifndef SDSRPCSERVER_H
#define SDSRPCSERVER_H

#include "localnode.h"

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
    static int ping(SdsRpcServer *srv, SdsBytesBuf &args, SdsBytesBuf &reply);
    static int findNode(SdsRpcServer *srv, SdsBytesBuf &args, SdsBytesBuf &reply);
    static int storeResult(SdsRpcServer *srv, SdsBytesBuf &args, SdsBytesBuf &reply);
    static int findResults(SdsRpcServer *srv, SdsBytesBuf &args, SdsBytesBuf &reply);

    bool running;

    LocalNode *localNode;
    std::deque<int> clientsQueue;
    pthread_mutex_t mutex;
    pthread_cond_t cond;
    pthread_t *threadPool;
    int server_fd;
};

#endif // SDSRPCSERVER_H
