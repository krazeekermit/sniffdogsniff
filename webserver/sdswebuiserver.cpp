#include "sdswebuiserver.h"

#include "common/loguru.hpp"
#include <dirent.h>

#include <vector>
#include <ostream>

/*
    Handlers
*/
class WebUiHandler : public HttpRequestHandler
{
public:
    WebUiHandler(LocalNode *node_, std::string &path)
        : node(node_)
    {}

    ~WebUiHandler()
    {}

    virtual HttpCode handleRequest(HttpRequest &request, HttpResponse &response) override
    {
        std::ostringstream ss;
        ss << "<!DOCTYPE html>";
        ss << "<html lang=\"en\" xmlns=\"http://www.w3.org/1999/html\" xmlns=\"http://www.w3.org/1999/html\" xmlns=\"http://www.w3.org/1999/html\">";
        ss << "<head>";
        ss << "<meta charset=\"UTF-8\"><title>SniffDigSniff - Search</title><style>";
        ss << ".text-center{text-align:center}";
        ss << ".container{width:100%;padding-right:15px;padding-left:15px;margin-right:auto;margin-left:auto}";
        ss << ".btn-gradient {background: linear-gradient(45deg, #979797, #858585 70%);color: #fff;padding: 6px 7px;border: none;border-radius: 2px;cursor: pointer;}";
        ss << ".sds-input {padding: 5px 7px;border: none;border-radius: 2px;border-style: solid;border-width: 0.5px;border-color: #858585;cursor: pointer;}";
        ss << ".sds-input-sub {width: 80%;padding: 5px 7px;border: none;border-radius: 2px;border-style: solid;border-width: 0.5px;border-color: #858585;cursor: pointer;}";
        ss << ".input-group{width:100%;height: 20px;display: inline-block;}";
        ss << ".input-group-append{display: inline-block;}";
        ss << ".sds-groupbox-first{margin-top: 15px;margin-bottom: 5px;padding-right:15px;padding-left:15px;}";
        ss << ".sds-groupbox-spacer{margin-left: 15px;margin-right: 15px;padding: auto;display: inline-block;}";
        ss << ".sds-groupbox-item{display: inline-block;}";
        ss << ".thing-align-right{float: right;}";
        ss << ".error-box{width: 100%; border-style: solid; border-width: 1px; border-color: red; color: red;}";
        ss << "</style></head>";
        ss << "<body class=\"\">";

        populateBody(request, ss);

        ss << "</body></html>";

        response.writeResponse(ss.str());
        return HttpCode::HTTP_OK;

    }

protected:
    virtual void populateBody(HttpRequest &request, std::ostringstream &ss) = 0;

    LocalNode *node;
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

    virtual void populateBody(HttpRequest &request, std::ostringstream &ss) override
    {
        ss << "<nav class=\"navbar navbar-light bg-light\">";
        ss << "<a class=\"navbar-brand\" href=\"/insert_link\">Insert link</a>";
        ss << "</nav>";
        ss << "<div class=\"container text-center\">";
        ss << "<div class=\"text-center\">";
        ss << "<img class=\"d-block mx-auto mb-4\" src=\"sds_header.png\" alt=\"\">";
        ss << "</div>";
        ss << "<form action=\"/search\" method=\"get\">";
        ss << "<div class=\"input-group\">";
        ss << "<input type=\"search\" class=\"sds-input\" placeholder=\"Search\" aria-label=\"Search\" aria-describedby=\"search-addon\" name=\"q\" value=\"Search something...\" style=\"width:50%\"/>";
        ss << "<div class=\"input-group-append\">";
        ss << "<button class=\"btn-gradient\" type=\"submit\">Search</button>";
        ss << "</div>";
        ss << "</div>";
        ss << "<div class=\"sds-groupbox-first\">";
        ss << "<div class=\"sds-groupbox-item\">";
        ss << "<label>Search on:</label>";
        ss << "</div>";
        ss << "<div class=\"sds-groupbox-item\">";
        ss << "<input class=\"form-check-input\" type=\"radio\" name=\"link_filter\" id=\"inlineRadio1\" value=\"all\" checked>";
        ss << "<label class=\"form-check-label\" for=\"inlineRadio1\">All links</label>";
        ss << "</div>";
        ss << "<div class=\"sds-groupbox-item\">";
        ss << "<input class=\"form-check-input\" type=\"radio\" name=\"link_filter\" id=\"inlineRadio2\" value=\"clearnet\">";
        ss << "<label class=\"form-check-label\" for=\"inlineRadio2\">Clear web links</label>";
        ss << "</div>";
        ss << "<div class=\"sds-groupbox-item\">";
        ss << "<input class=\"form-check-input\" type=\"radio\" name=\"link_filter\" id=\"inlineRadio3\" value=\"onion\">";
        ss << "<label class=\"form-check-label\" for=\"inlineRadio3\">Onion links</label>";
        ss << "</div>";
        ss << "<div class=\"sds-groupbox-item\">";
        ss << "<input class=\"form-check-input\" type=\"radio\" name=\"link_filter\" id=\"inlineRadio4\" value=\"i2p\">";
        ss << "<label class=\"form-check-label\" for=\"inlineRadio4\">I2P links</label>";
        ss << "</div>";
        ss << "<div  class=\"sds-groupbox-spacer\">";
        ss << "<label>|</label>";
        ss << "</div>";
        ss << "<div class=\"sds-groupbox-item\">";
        ss << "<label>Category:</label>";
        ss << "</div>";
        ss << "<div class=\"sds-groupbox-item\">";
        ss << "<input class=\"form-check-input\" type=\"radio\" name=\"category\" id=\"inlineRadio1\" value=\"all\" checked>";
        ss << "<label class=\"form-check-label\" for=\"inlineRadio1\">All</label>";
        ss << "</div>";
        ss << "<div class=\"sds-groupbox-item\">";
        ss << "<input class=\"form-check-input\" type=\"radio\" name=\"category\" id=\"inlineRadio2\" value=\"links\">";
        ss << "<label class=\"form-check-label\" for=\"inlineRadio2\">Links</label>";
        ss << "</div>";
        ss << "<div class=\"sds-groupbox-item\">";
        ss << "<input class=\"form-check-input\" type=\"radio\" name=\"category\" id=\"inlineRadio3\" value=\"images\">";
        ss << "<label class=\"form-check-label\" for=\"inlineRadio3\">Images</label>";
        ss << "</div>";
        ss << "<div class=\"sds-groupbox-item\">";
        ss << "<input class=\"form-check-input\" type=\"radio\" name=\"category\" id=\"inlineRadio4\" value=\"videos\">";
        ss << "<label class=\"form-check-label\" for=\"inlineRadio4\">Videos</label>";
        ss << "</div>";
        ss << "</div>";
        ss << "<input type=\"hidden\" name=\"data_type\" value=\"links\"/>";
        ss << "</form>";
    }
};

class ResultsViewHandler : public WebUiHandler
{

public:
    ResultsViewHandler(LocalNode *node_, std::string path)
        : WebUiHandler(node_, path) {}

    virtual void populateBody(HttpRequest &request, std::ostringstream &ss) override
    {
        std::string query = request.values["q"];
        if (request.values["link_filter"] != this->linkFilter) {
            this->linkFilter = request.values["link_filter"];
        }
        if (request.values["data_type"] != this->dataType) {
            this->dataType = request.values["data_type"];
        }
        std::vector<SearchEntry> results;
        this->node->doSearch(results, query.c_str());

        ss << "<div class=\"fixed-top \">";
        ss << "<nav class=\"navbar navbar-light bg-light\">";
        ss << "<form action=\"/search\" method=\"get\">";
        ss << "<a class=\"navbar-brand\" href=\"/\">";
        ss << "<img src=\"sds_logo.png\" width=\"40\" height=\"40\"> SniffDogSniff </a>";
        ss << "<div class=\"input-group\">";
        ss << "<input type=\"search\" class=\"sds-input\" placeholder=\"Search\" aria-label=\"Search\" aria-describedby=\"search-addon\" name=\"q\" value=\"Search something...\" style=\"width:50%\"/>";
        ss << "<div class=\"input-group-append\">";
        ss << "<button class=\"btn-gradient\" type=\"submit\">Search</button>";
        ss << "</div></div>";
        ss << "<div class=\"sds-groupbox-first\">";
        ss << "<div class=\"sds-groupbox-item\">";
        ss << "<label>Search on:</label>";
        ss << "</div>";
        ss << "<div class=\"sds-groupbox-item\">";
        ss << "<input class=\"form-check-input\" type=\"radio\" name=\"link_filter\" id=\"inlineRadio1\" value=\"all\" checked>";
        ss << "<label class=\"form-check-label\" for=\"inlineRadio1\">All links</label>";
        ss << "</div>";
        ss << "<div class=\"sds-groupbox-item\">";
        ss << "<input class=\"form-check-input\" type=\"radio\" name=\"link_filter\" id=\"inlineRadio2\" value=\"clearnet\">";
        ss << "<label class=\"form-check-label\" for=\"inlineRadio2\">Clear web links</label>";
        ss << "</div>";
        ss << "<div class=\"sds-groupbox-item\">";
        ss << "<input class=\"form-check-input\" type=\"radio\" name=\"link_filter\" id=\"inlineRadio3\" value=\"onion\">";
        ss << "<label class=\"form-check-label\" for=\"inlineRadio3\">Onion links</label>";
        ss << "</div>";
        ss << "<div class=\"sds-groupbox-item\">";
        ss << "<input class=\"form-check-input\" type=\"radio\" name=\"link_filter\" id=\"inlineRadio4\" value=\"i2p\">";
        ss << "<label class=\"form-check-label\" for=\"inlineRadio4\">I2P links</label>";
        ss << "</div>";
        ss << "<div  class=\"sds-groupbox-spacer\">";
        ss << "<label>|</label>";
        ss << "</div>";
        ss << "<div class=\"sds-groupbox-item\">";
        ss << "<label>Category:</label>";
        ss << "</div>";
        ss << "<div class=\"sds-groupbox-item\">";
        ss << "<input class=\"form-check-input\" type=\"radio\" name=\"category\" id=\"inlineRadio1\" value=\"all\" checked>";
        ss << "<label class=\"form-check-label\" for=\"inlineRadio1\">All</label>";
        ss << "</div>";
        ss << "<div class=\"sds-groupbox-item\">";
        ss << "<input class=\"form-check-input\" type=\"radio\" name=\"category\" id=\"inlineRadio2\" value=\"links\">";
        ss << "<label class=\"form-check-label\" for=\"inlineRadio2\">Links</label>";
        ss << "</div>";
        ss << "<div class=\"sds-groupbox-item\">";
        ss << "<input class=\"form-check-input\" type=\"radio\" name=\"category\" id=\"inlineRadio3\" value=\"images\">";
        ss << "<label class=\"form-check-label\" for=\"inlineRadio3\">Images</label>";
        ss << "</div>";
        ss << "<div class=\"sds-groupbox-item\">";
        ss << "<input class=\"form-check-input\" type=\"radio\" name=\"category\" id=\"inlineRadio4\" value=\"videos\">";
        ss << "<label class=\"form-check-label\" for=\"inlineRadio4\">Videos</label>";
        ss << "</div>";
        ss << "</div>";
        ss << "<input type=\"hidden\" name=\"data_type\" value=\"links\"/>";
        ss << "</form>";
        ss << "</nav>";
        ss << "</div>";
        ss << "<hr/>";
        ss << "<main class=\"container\">";
        ss << "<div class=\"col-14 mx-auto\">";
        ss << "<ul class=\"list-group list-group-flush\">";

        for (auto rit = results.begin(); rit < results.end(); rit++) {
            ss << "<li class=\"list-group-item\">";
            ss << "<div>";
            ss << "<small class=\"disabled\">" << rit->getUrl() << "</small>";
            ss << "<small class=\"float-right\">";
            // ss << "<a class=\"btn btn btn-outline-secondary btn-sm\" role=\"button\" href=\"/invalidate?hash=\" data-toggle=\"tooltip\" data-placement=\"top\" title=\"Mark link for removal: if in your opinion this link contains offensive content (see offensive.md) you can mark it for removal\">":
            ss << "Mark for removal";
            ss << "</a>";
            ss << "</small>";
            ss << "<br/>";
            ss << "<img src=\"\" width=\"16\" height=\"16\">";
            ss << "<a href=\"" << rit->getUrl() << "\">" << rit->getTitle() << "</a>";
            ss << "<p class=\"mb-1\">" /* << rit->getProperty(DESCRIPTION) */<< "</p>";
            ss << "</div>";
            ss << "</li>";
        }

        ss << "</ul>";
        ss << "</div>";
        ss << "</main>";
    }

private:
    std::string linkFilter;
    std::string dataType;
};

class InsertEntryHandler : public WebUiHandler
{

public:
    InsertEntryHandler(LocalNode *node_, std::string path)
        : WebUiHandler(node_, path) {}

    virtual void populateBody(HttpRequest &request, std::ostringstream &ss) override
    {
        ss << "<main><h2>Insert Link</h2></div>";
        for (auto it = request.values.begin(); it != request.values.end(); it++) LOG_S(2) <<"VALS:: "<< it->first <<":"<< it->second;
        if (!request.values.empty()) {
            ss << "<div class=\"error-box\"><p>Result insertion error: unknown<p></div>";
        }
        ss << "<form class=\"container\" action=\"/insert_link\" method=\"post\">";
        ss << "<p>Link title:</p>";
        ss << "<input type=\"text\" class=\"sds-input-sub\" id=\"text_input_title\" placeholder=\"Title\" aria-label=\"Search\" name=\"link_title\"/>";
        ss << "<p>Link url</p>";
        ss << "<input type=\"text\" class=\"sds-input-sub\" id=\"text_input_url\" placeholder=\"http://example.com\" aria-label=\"Search\" name=\"link_url\"/>";
        ss << "<p>Link description</p>";
        ss << "<textarea rows=\"4\" class=\"sds-input-sub\" id=\"text_area_description\" placeholder=\"Something...\" aria-label=\"Search\" name=\"link_description\"></textarea>";
        ss << "<p></p><label>Link Type (cathegory) </label>";
        ss << "<select class=\"form-control\" id=\"data_type_combo\" name=\"data_type\">";
        ss << "<option value=\"links\">Link</option>";
        ss << "<option value=\"images\">Image</option>";
        ss << "<option value=\"videos\">Video</option>";
        ss << "</select>";
        ss << "<input type=\"submit\" class=\"btn-gradient thing-align-right\" value=\" Insert \"/></form></main>";
    };
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
    this->addHandler("/insert_link", new InsertEntryHandler(this->node, ""));

    this->addHandler("/sds_logo.png", new FileHandler(this->resourcesDir, "image/png"));
    this->addHandler("/sds_header.png", new FileHandler(this->resourcesDir, "image/png"));
}
