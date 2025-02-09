#ifndef SDSWEBUISERVER_H
#define SDSWEBUISERVER_H

#include "httpserver.h"
#include "Jinja2CppLight.h"

#include "sds_core/localnode.h"

#include <functional>

class SdsWebUiServer : public HttpServer
{
    friend class IndexHandler;
    friend class FileHandler;
    friend class ResultsViewHandler;
    friend class InserResultsHandler;

public:
    SdsWebUiServer(LocalNode *node_, std::string resourcesDir_);
    ~SdsWebUiServer();

private:
    LocalNode *node;
    std::string resourcesDir;

    void createHandlers();
};

#endif // SDSWEBUISERVER_H
