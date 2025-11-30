#ifndef SDSWEBUISERVER_H
#define SDSWEBUISERVER_H

#include "httpserver.h"

#include "sds_core/localnode.h"

#include <functional>

class SdsWebUiServer : public HttpServer
{
public:
    SdsWebUiServer(LocalNode *node_, std::string resourcesDir_);
    ~SdsWebUiServer();

private:
    LocalNode *node;
    std::string resourcesDir;

    void createHandlers();
};

#endif // SDSWEBUISERVER_H
