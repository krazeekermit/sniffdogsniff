#include "httpserver.h"

#include "logging.hpp"

#include <sys/socket.h>
#include <arpa/inet.h>
#include <unistd.h>

#include <cstring>

#define THREAD_POOL_SZ 1

//static void httpUnescape(std::string &ss)
//{
//    int i;
//    std::string unescaped = "";
//    for (i = 0; i < ss.length(); i++) {

//    }
//}

static int parseHttpAttrs(HttpRequest &req, char *valssp)
{
    if (!strlen(valssp))
        return 0;

    req.values.clear();
    char *linestart = nullptr;
    char *lineend = nullptr;
    char *keyp = nullptr;
    char *valuep = nullptr;
    linestart = strtok_r(valssp, "&", &lineend);
    do {
        keyp = strtok_r(linestart, "=", &valuep);
        if (strlen(keyp)) {
            if (!strlen(valuep))
                return -1; //BAD REQUEST

            req.values.emplace(keyp, valuep);
        }
    } while ((linestart = strtok_r(nullptr, "&", &lineend)) != nullptr);
    return 0;
}

static int parseHttpFirstLine(HttpRequest &req, char *line)
{
    char *linestart = nullptr;
    char *lineend = nullptr;
    char *lineend2 = nullptr;
    linestart = strtok_r(line, " ", &lineend);
    if (strcmp(linestart, "GET") == 0) {
        req.method = HttpMethod::HTTP_GET;
        linestart = strtok_r(nullptr, " ", &lineend);
        linestart = strtok_r(linestart, "?", &lineend2);
        req.url = linestart;
        if (parseHttpAttrs(req, lineend2))
            return -1;

    } else if (strcmp(linestart, "POST") == 0) {
        req.method = HttpMethod::HTTP_POST;
        req.url = strtok_r(nullptr, " ", &lineend);
    }

    req.version = strtok_r(nullptr, " ", &lineend);

    return 0;
}

static int parseHttpReqest(HttpRequest &req, std::string &ss)
{
    int ret = 0;
    size_t slen = ss.length();
    char *parsep = new char[slen + 1];

    size_t i;
    size_t parsep_len = 0;
    for (i = 0; i < slen; i++) {
        if (ss[i] != '\r')
            parsep[parsep_len++] = ss[i];
    }
    parsep[parsep_len] = '\0';

    char *linestart = nullptr;
    char *lineend = nullptr;
    char *keyp = nullptr;
    char *valuep = nullptr;
    linestart = strtok_r(parsep, "\n", &lineend);
    if (parseHttpFirstLine(req, linestart)) {
        ret = -1;
        goto parse_end;
    }

    req.headers.clear();
    while ((linestart = strtok_r(nullptr, "\n", &lineend)) != nullptr) {
        if (strlen(linestart)) {
            keyp = strtok_r(linestart, ":", &valuep);
            req.headers.emplace(keyp, valuep);
        }
    }

    if (req.method == HttpMethod::HTTP_POST) {
        if (parseHttpAttrs(req, lineend)) {
            ret = -1;
            goto parse_end;
        }
    }

    ret = 0;

parse_end:
    delete[] parsep;
    return ret;
}

static int createResponse(HttpResponse &resp, std::string &body)
{
    body.clear();
    body += "HTTP/1.1 " + std::to_string(resp.code) + " Created\r\n";
    for (auto it = resp.headers.begin(); it != resp.headers.end(); it++)
        body += it->first + ":" + it->second + "\r\n";

    body += "\r\n\r\n";
    body += resp.ss;
    return 0;
}

HttpServer::HttpServer()
    : defaultContentType("text/html")
{
    pthread_mutex_init(&this->mutex, nullptr);
    pthread_cond_init(&this->cond, nullptr);
}

HttpServer::~HttpServer()
{
    for (auto it = this->handlers.begin(); it != this->handlers.end(); it++)
        delete it->second;

    this->handlers.clear();
}

void *accessHandlerCallback(void *cls) {
    HttpServer *srv = static_cast<HttpServer*>(cls);
    int client_fd;
    int previous_error = 0;
    if (srv) {
        while (1) {
            pthread_mutex_lock(&srv->mutex);
            while (srv->clientsQueue.empty()) {
                pthread_cond_wait(&srv->cond, &srv->mutex);
            }
            client_fd = srv->clientsQueue.front();
            srv->clientsQueue.pop_front();
            pthread_mutex_unlock(&srv->mutex);

            char buf[512];
            size_t nrecv = 0;
            size_t trecv = 0;
            std::string ss;
            while ((nrecv = recv(client_fd, buf, sizeof(buf), 0))) {
                trecv += nrecv;
                ss.append(buf, nrecv);
                if (nrecv < sizeof(buf))
                    break;
            }
            ss.append("\0");

            HttpRequest request;
            HttpResponse response;
            if (parseHttpReqest(request, ss)) {

            }

            previous_error = srv->handleRequest(request, response);

            std::string respBody = "";
            createResponse(response, respBody);

            send(client_fd, respBody.data(), respBody.length(), 0);
            close(client_fd);
        }
    }
    return 0;
}

int HttpServer::startListening(const char *addrstr, int port)
{
    int i, fd, client_fd;
    ssize_t valread;
    struct sockaddr_in address;
    int opt = 1;
    socklen_t addrlen = sizeof(address);

    if (!this->threadPool) {
        this->threadPool = new pthread_t[THREAD_POOL_SZ];
        for (i = 0; i < THREAD_POOL_SZ; i++) {
            pthread_create(&this->threadPool[i], nullptr, &accessHandlerCallback, this);
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
        logdebug << "COND >>>>>>>>>>>>><|||\n";
        this->clientsQueue.push_back(client_fd);
        pthread_cond_signal(&this->cond);
    }

    return 0;
}

int HttpServer::shutdown()
{

}

void HttpServer::addHandler(std::string u, HttpRequestHandler *h)
{
    this->handlers[u] = h;
}

void HttpServer::removeHandler(std::string u)
{
    this->handlers.erase(u);
}

bool HttpServer::hasHandlerFor(std::string u)
{
    return this->handlers.find(u) != this->handlers.end();
}

int HttpServer::handleRequest(HttpRequest &request, HttpResponse &response)
{
    int code = 201;

    response.headers["Content-Type"] = this->defaultContentType;
    if (this->hasHandlerFor(request.url)) {
        code = this->handlers[request.url]->handleRequest(request, response);
    } else {
        code = this->handleError(request, response, 404);
    }

    return code;
}

int HttpServer::handleError(HttpRequest &request, HttpResponse &response, int errorCode)
{
    response.ss += "<html><body>";
    response.ss += "<h2> Error " + std::to_string(errorCode) + "</h2>";
    switch (errorCode) {
    case 404:
        response.ss += "<h4>" + request.url + " not found" + "</h4>";
        break;
    default:
        break;
    }
    response.ss + "</body></html>";
    return errorCode;
}
