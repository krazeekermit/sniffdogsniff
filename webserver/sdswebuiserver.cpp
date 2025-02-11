#include "sdswebuiserver.h"

#include "common/logging.h"

#include "dirent.h"

#include <vector>

/*
    Handlers
*/
class WebUiHandler : public HttpRequestHandler
{
public:
    WebUiHandler(LocalNode *node_, std::string &path)
        : node(node_)
    {
        FILE *fp = fopen(path.c_str(), "rb");
        if (fp) {
            std::string ss = "";
            char buf[1024];
            size_t nread = 0;
            while ((fgets(buf, sizeof(buf), fp) != nullptr)) {
                ss += buf;
            }

            this->templ = new Jinja2CppLight::Template(ss);
            fclose(fp);
        }
    }

    ~WebUiHandler()
    {
        delete templ;
    }

protected:
    LocalNode *node;
    Jinja2CppLight::Template *templ;
};

class FileHandler : public HttpRequestHandler
{

public:
    FileHandler(std::string &resourcesDir_, std::string contentType_)
        : resourcesDir(resourcesDir_), contentType(contentType_) {}

    virtual HttpCode handleRequest(HttpRequest &request, HttpResponse &response) override
    {
        char path[1024];
        sprintf(path, "%s/static/%s", this->resourcesDir.c_str(), request.url.c_str());
        FILE *fp = fopen(path, "rb");
        if (!fp)
            return HttpCode::HTTP_NOT_FOUND;

        uint8_t buf[1024];
        size_t nread = 0;
        while ((nread = fread(buf, sizeof(uint8_t), 1024, fp)) > 0) {
            response.buffer.writeBytes(buf, nread);
        }

        fclose(fp);

        response.headers["Content-Type"] = contentType;
        response.headers["Content-Length"] = std::to_string(response.buffer.size());
        return HttpCode::HTTP_OK;
    }

private:
    std::string resourcesDir;
    std::string contentType;
};

class IndexHandler : public WebUiHandler
{

public:
    IndexHandler(LocalNode *node_, std::string path)
        : WebUiHandler(node_, path) {}

    virtual HttpCode handleRequest(HttpRequest &request, HttpResponse &response) override
    {
        std::string ss = this->templ->render();
        response.buffer.writeBytes((unsigned char*) ss.c_str(), ss.length());
        return HttpCode::HTTP_CREATED;
    }
};

class ResultsViewHandler : public WebUiHandler
{

public:
    ResultsViewHandler(LocalNode *node_, std::string path)
        : WebUiHandler(node_, path) {}

    virtual HttpCode handleRequest(HttpRequest &request, HttpResponse &response) override
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
        this->node->doSearch(results, query.c_str());
        Jinja2CppLight::Template *templ = this->templ;

        templ->setValue("q", query);

        Jinja2CppLight::TupleValue ress;
        for (auto it = results.begin(); it != results.end(); it++)
            ress.addValue(Jinja2CppLight::TupleValue::create(it->getTitle(), it->getUrl(), ""));

        templ->setValue("results", ress);

        try {
            std::string ss = templ->render();
            response.buffer.writeBytes((unsigned char*) ss.c_str(), ss.length());
        } catch (std::exception &ex) {
            logerr << "webui template error: " << ex.what();
            return HttpCode::HTTP_INTERNAL_ERROR;
        }

        return HttpCode::HTTP_CREATED;
    }

private:
    std::string linkFilter;
    std::string dataType;
};

/* Web UI Server */

SdsWebUiServer::SdsWebUiServer(LocalNode *node_, std::string resourcesDir_)
    : node(node_), resourcesDir(resourcesDir_)
{
    this->createHandlers();
}

SdsWebUiServer::~SdsWebUiServer()
{
}

void SdsWebUiServer::createHandlers()
{
    this->addHandler("/", new IndexHandler(this->node, this->resourcesDir + "/templates/index.html"));
    this->addHandler("/search", new ResultsViewHandler(this->node, this->resourcesDir + "/templates/results_links.html"));

    this->addHandler("/style.css", new FileHandler(this->resourcesDir, "text/css"));
    this->addHandler("/sds_logo.png", new FileHandler(this->resourcesDir, "image/png"));
    this->addHandler("/sds_header.png", new FileHandler(this->resourcesDir, "image/png"));
}
