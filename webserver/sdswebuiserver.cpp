#include "sdswebuiserver.h"

#include "dirent.h"

#include <vector>

static int loadFile(const char *path, std::string &ss)
{
    FILE *fp = fopen(path, "rb");
    if (!fp)
        return -1;

    char buf[1024];
    size_t nread = 0;
    while ((nread = fread(buf, sizeof(char), 1024, fp)) > 0) {
        ss.append(buf, nread);
    }

    fclose(fp);
    return 0;
}

/*
    Handlers
*/
class WebUiHandler : public HttpRequestHandler {
public:
    WebUiHandler(SdsWebUiServer *srv_)
        : srv(srv_) {}

protected:
    SdsWebUiServer *srv;
};

class IndexHandler : public WebUiHandler {

public:
    IndexHandler(SdsWebUiServer *srv_)
        : WebUiHandler(srv_) {}

    virtual int handleRequest(HttpRequest &request, HttpResponse &response) override
    {
        response.ss = this->srv->templates["index.html"]->render();
        return 201;
    }
};

class FileHandler : public WebUiHandler {

public:
    FileHandler(SdsWebUiServer *srv_, std::string contentType_)
        : WebUiHandler(srv_), contentType(contentType_) {}

    virtual int handleRequest(HttpRequest &request, HttpResponse &response) override
    {
        char path[1024];
        sprintf(path, "%s/static/%s", this->srv->resourcesDir.c_str(), request.url.c_str());
        response.headers["Content-Type"] = contentType;
        if (loadFile(path, response.ss))
            return 404;

        return 201;
    }

private:
    std::string contentType;
};


class ResultsViewHandler : public WebUiHandler {

public:
    ResultsViewHandler(SdsWebUiServer *srv_)
        : WebUiHandler(srv_) {}

    virtual int handleRequest(HttpRequest &request, HttpResponse &response) override
    {
        //q=Search+something...&link_filter=all&data_type=links
        std::string query = request.values["q"];
        if (request.values["link_filter"] != this->linkFilter) {
            this->linkFilter = request.values["link_filter"];
        }
        if (request.values["data_type"] != this->dataType) {
            this->dataType = request.values["data_type"];
        }
        std::vector<SearchEntry> results;
        this->srv->node->doSearch(results, query.c_str());
        Jinja2CppLight::Template *templ = this->srv->templates["results_links.html"];

        templ->setValue("q", query);

        Jinja2CppLight::TupleValue ress;
        for (auto it = results.begin(); it != results.end(); it++)
            ress.addValue(Jinja2CppLight::TupleValue::create(it->getTitle(), it->getUrl(), ""));

        templ->setValue("results", ress);

        try {
            response.ss = templ->render();
        } catch (Jinja2CppLight::render_error &ex) {
            std::cerr << "template error ::::" << ex.what();
        }

        return 201;
    }

private:
    std::string linkFilter;
    std::string dataType;
};

/* Web UI Server */

SdsWebUiServer::SdsWebUiServer(LocalNode *node_, std::string resourcesDir_)
    : node(node_), resourcesDir(resourcesDir_)
{
    this->loadTemplates();
    this->createHandlers();
}

SdsWebUiServer::~SdsWebUiServer()
{
    for (auto it = this->templates.begin(); it != this->templates.end(); it++)
        delete it->second;

    this->templates.clear();
}

void SdsWebUiServer::loadTemplates()
{
    char path[1024];
    sprintf(path, "%s/templates/", this->resourcesDir.c_str());

    DIR *dir;
    struct dirent *ent;
    if ((dir = opendir(path))) {
        while ((ent = readdir(dir))) {
            std::string ss = "";
            sprintf(path, "%s/templates/%s", this->resourcesDir.c_str(), ent->d_name);

            if (loadFile(path, ss)) {
                continue;
            }

            this->templates[ent->d_name] = new Jinja2CppLight::Template(ss);
        }
    }
}

void SdsWebUiServer::createHandlers()
{
    this->addHandler("/", new IndexHandler(this));
    this->addHandler("/style.css", new FileHandler(this, "text/css"));

    this->addHandler("/search", new ResultsViewHandler(this));
}

int SdsWebUiServer::handleRequest(HttpRequest &request, HttpResponse &response)
{
    return HttpServer::handleRequest(request, response);
}
