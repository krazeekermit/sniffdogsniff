#ifndef SDSP2PSERVER_H
#define SDSP2PSERVER_H

#include "sds_core/localnode.h"

class SdsP2PServer
{
public:
    SdsP2PServer(LocalNode *node);
    ~SdsP2PServer();

    int startListening(const char *addrstr, int port);
    void shutdown();

private:
    void handleRequest(int client_fd);
    int ping(SdsBytesBuf &args, SdsBytesBuf &reply);
    int findNode(SdsBytesBuf &args, SdsBytesBuf &reply);
    int storeResult(SdsBytesBuf &args, SdsBytesBuf &reply);
    int findResults(SdsBytesBuf &args, SdsBytesBuf &reply);

    bool running;

    LocalNode *localNode;
    int server_fd;
};

#endif // SDSRPCSERVER_H
