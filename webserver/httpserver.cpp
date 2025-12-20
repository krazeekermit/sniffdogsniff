#include "httpserver.h"

#include "common/loguru.hpp"
#include "common/stringutil.h"

#include <pthread.h>
#include <sys/socket.h>
#include <arpa/inet.h>
#include <unistd.h>

#include <cstring>

#define THREAD_POOL_SZ 1

static std::string httpUnescape(const std::string ss)
{
    std::string unescaped = "";

    char c, hi, lo;
    int i = 0;
    while (i < ss.length()) {
        c = ss[i++];
        switch (c) {
        case '+':
            unescaped += ' ';
            break;
        case '%':
        case '$':
            hi = ss[i++];
            lo = ss[i++];
            if (hi >= '0' && hi <= '9')
                hi = hi - '0';
            else if (hi >= 'A' && hi <= 'F')
                hi = hi - 'A' + 10;

            if (lo >= '0' && lo <= '9')
                lo = lo - '0';
            else if (lo >= 'A' && lo <= 'F')
                lo = lo - 'A' + 10;

            c = ((hi << 4) & 0xf0) | (lo & 0x0f);
            unescaped += c;
            break;
        default:
            unescaped += c;
        }
    }
    return unescaped;
}

static int parseHttpAttrs(HttpRequest &req, std::string &attrsString)
{
    req.headers.clear();
    std::vector<std::string> toks = split(attrsString, "&");
    for (auto it = toks.begin(); it != toks.end(); it++) {
        ssize_t delimPos = it->find("=");
        if (delimPos == std::string::npos) {
            return -1;
        }

        req.values[it->substr(0, delimPos)] = httpUnescape(it->substr(delimPos +1));
    }

    return 0;
}

static int parseHttpHeaders(HttpRequest &req, std::string &headersString)
{
    req.values.clear();
    std::vector<std::string> toks = split(headersString, "\r\n");
    for (auto it = toks.begin(); it != toks.end(); it++) {
        ssize_t delimPos = it->find(":");
        if (delimPos == std::string::npos) {
            return -1;
        }

        req.headers[it->substr(0, delimPos)] = it->substr(delimPos +1);
    }

    return 0;
}

static HttpCode parseHttpFirstLine(HttpRequest &req, std::string &line)
{
    LOG_F(1, "new http request: %s", line.c_str());
    std::vector<std::string> toks = split(line, " ");
    if (toks.size() > 2) {
        req.url = toks[1];
        if (toks[0] == "GET" || toks[0] == "HEAD") {
            req.method = toks[0] == "GET" ? HttpMethod::HTTP_GET : HttpMethod::HTTP_HEAD;
            ssize_t attrsPos = toks[1].find("?");
            if (attrsPos != std::string::npos) {
                req.url = toks[1].substr(0, attrsPos);
                std::string attrsString = toks[1].substr(attrsPos + 1);
                if (parseHttpAttrs(req, attrsString)) {
                    LOG_F(ERROR, "bad http request url %s", attrsString.c_str());
                    return HttpCode::HTTP_BAD_REQUEST;
                }
            }
        } else if (toks[0] == "POST") {
            req.method = HttpMethod::HTTP_POST;
        } else {
            return HttpCode::HTTP_NOT_IMPLEMENTED;
        }

        req.version = toks[2];

        return HttpCode::HTTP_OK;
    }
    return HttpCode::HTTP_BAD_REQUEST;
}

static HttpCode parseHttpReqest(HttpRequest &req, std::string &sbuffer)
{
    ssize_t lineStart = 0;
    ssize_t lineEnd = sbuffer.find("\r\n");
    if (lineEnd == std::string::npos) {
        return HttpCode::HTTP_BAD_REQUEST;
    }

    std::string line = sbuffer.substr(lineStart, lineEnd);
    lineStart = lineEnd + 2;
    if (parseHttpFirstLine(req, line) != HttpCode::HTTP_OK) {
        return HttpCode::HTTP_BAD_REQUEST;
    }

    lineEnd = sbuffer.find("\r\n\r\n", lineStart);
    if (lineEnd != std::string::npos) {
        std::string headersStr = sbuffer.substr(lineStart, lineEnd - lineStart);
        if (parseHttpHeaders(req, headersStr)) {
            return HttpCode::HTTP_BAD_REQUEST;
        }

        req.data = sbuffer.substr(lineEnd + 4);
        if (req.method == HttpMethod::HTTP_POST) {
            if (parseHttpAttrs(req, req.data)) {
                return HttpCode::HTTP_BAD_REQUEST;
            }
        }
    }

    return HttpCode::HTTP_OK;
}

static int createResponse(HttpResponse &resp, std::string &sbuffer)
{
    sbuffer.clear();
    sbuffer += "HTTP/1.1 " + std::to_string(resp.code) + "\r\n";
    for (auto it = resp.headers.begin(); it != resp.headers.end(); it++)
        sbuffer += it->first + ":" + it->second + "\r\n";

    sbuffer += "\r\n";
    sbuffer += resp.buffer;
    return 0;
}

HttpServer::HttpServer()
    : detach(false), defaultContentType("text/html")
{
    pthread_mutex_init(&this->mutex, nullptr);
    pthread_cond_init(&this->cond, nullptr);
}

HttpServer::~HttpServer()
{
    this->shutdown();
    pthread_mutex_destroy(&this->mutex);
    pthread_cond_destroy(&this->cond);

    for (auto it = this->handlers.begin(); it != this->handlers.end(); it++)
        delete it->second;

    this->handlers.clear();
}

void *accessHandlerCallback(void *cls) {
    HttpServer *srv = static_cast<HttpServer*>(cls);
    int client_fd;
    HttpCode httpErr = HttpCode::HTTP_OK;
    if (srv) {
        while (srv->running) {
            pthread_mutex_lock(&srv->mutex);
            while (srv->clientsQueue.empty()) {
                if (!srv->running) {
                    pthread_mutex_unlock(&srv->mutex);
                    return nullptr;
                }
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

            HttpRequest request;
            HttpResponse response;
            httpErr = parseHttpReqest(request, ss);
            if (httpErr == HttpCode::HTTP_OK) {
                httpErr = srv->handleRequest(request, response);
            }

            if (httpErr != HttpCode::HTTP_OK) {
                srv->handleError(request, response, httpErr);
            }

            std::string respBody;
            createResponse(response, respBody);

            send(client_fd, respBody.data(), respBody.size(), 0);
            close(client_fd);
        }
    }
    return 0;
}

void *acceptFun(void *cls)
{
    HttpServer *srv = static_cast<HttpServer*>(cls);
    if (srv) {
        int client_fd;
        struct sockaddr_in address;
        socklen_t addrlen = sizeof(address);

        while ((client_fd = accept(srv->server_fd, (struct sockaddr*)&address, &addrlen)) > -1) {
            pthread_mutex_lock(&srv->mutex);
            LOG_F(1, "new connection request from %s:%d", inet_ntoa(address.sin_addr), ntohs(address.sin_port));
            srv->clientsQueue.push_back(client_fd);
            pthread_cond_signal(&srv->cond);
            pthread_mutex_unlock(&srv->mutex);
        }
    }

    return nullptr;
}

int HttpServer::startListening(const char *addrstr, int port, bool detach)
{
    int i, fd;
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
    if (listen(fd, 5) < 0) {
        return -2;
    }

    this->server_fd = fd;
    if (!this->threadPool) {
        this->running = true;
        this->threadPool = new pthread_t[THREAD_POOL_SZ];
        for (i = 0; i < THREAD_POOL_SZ; i++) {
            pthread_create(&this->threadPool[i], nullptr, &accessHandlerCallback, this);
        }
    }

    if (detach) {
        pthread_create(&this->acceptThread, nullptr, &acceptFun, this);
    } else {
        acceptFun(this);
    }

    return 0;
}

int HttpServer::shutdown()
{
    if (this->threadPool) {
        pthread_mutex_lock(&this->mutex);
        this->running = false;
        pthread_cond_broadcast(&this->cond);
        pthread_mutex_unlock(&this->mutex);

        int i;
        void *dummy = nullptr;
        for (i = 0; i < THREAD_POOL_SZ; i++) {
            pthread_join(this->threadPool[i], &dummy);
        }

        close(this->server_fd);
        pthread_cancel(this->acceptThread);

        delete[] this->threadPool;
        this->threadPool = nullptr;
    }
    return 0;
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

HttpCode HttpServer::handleRequest(HttpRequest &request, HttpResponse &response)
{
    response.headers["Content-Type"] = this->defaultContentType;
    if (this->hasHandlerFor(request.url)) {
        return this->handlers[request.url]->handleRequest(request, response);
    }

    LOG_F(1, "no handlerfor request %s", request.url.c_str());
    return HttpCode::HTTP_NOT_FOUND;
}

HttpCode HttpServer::handleError(HttpRequest &request, HttpResponse &response, HttpCode errorCode)
{
    response.buffer = "";
    response.buffer += "<html><body>";
    response.buffer += "<h2> Error " + std::to_string(errorCode) + "</h2>";
    switch (errorCode) {
    case HttpCode::HTTP_NOT_FOUND:
        response.buffer += "<h4>" + request.url + " Not Found" + "</h4>";
        break;
    case HttpCode::HTTP_INTERNAL_ERROR:
        response.buffer += "<h4> Internal Server Error </h4>";
        break;
    case HttpCode::HTTP_NOT_IMPLEMENTED:
        response.buffer += "<h4> Method Not Implemented </h4>";
        break;
    default:
        break;
    }
    response.buffer + "</body></html>";
    return errorCode;
}
