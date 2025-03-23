#include "httpserver.h"

#include <pthread.h>
#include <sys/socket.h>
#include <arpa/inet.h>
#include <unistd.h>

#include <cstring>
#include <poll.h>

#define MAX_CLIENT_COUNT 9
#define MAX_POLL_FD_COUNT (MAX_CLIENT_COUNT+1)

static std::string httpUnescape(const char *ss)
{
    std::string unescaped = "";

    char c, hi, lo;
    int i = 0;
    while (i < strlen(ss)) {
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

static HttpCode parseHttpFirstLine(HttpRequest &req, char *line)
{
    char *linestart = nullptr;
    char *lineend = nullptr;
    char *lineend2 = nullptr;
    linestart = strtok_r(line, " ", &lineend);
    if (strcmp(linestart, "GET") == 0 || strcmp(linestart, "HEAD")) {
        req.method = strcmp(linestart, "GET") ? HttpMethod::HTTP_GET : HttpMethod::HTTP_HEAD;
        linestart = strtok_r(nullptr, " ", &lineend);
        linestart = strtok_r(linestart, "?", &lineend2);
        req.url = linestart;
        if (parseHttpAttrs(req, lineend2))
            return HttpCode::HTTP_BAD_REQUEST;

    } else if (strcmp(linestart, "POST") == 0) {
        req.method = HttpMethod::HTTP_POST;
        req.url = strtok_r(nullptr, " ", &lineend);
    } else {
        return HttpCode::HTTP_NOT_IMPLEMENTED;
    }

    req.version = strtok_r(nullptr, " ", &lineend);

    return HttpCode::HTTP_OK;
}

static HttpCode parseHttpReqest(HttpRequest &req, SdsBytesBuf &sbuffer)
{
    HttpCode ret = HttpCode::HTTP_OK;

    size_t i;
    size_t parsep_len = 0;
    for (i = 0; i < sbuffer.size(); i++) {
        if (sbuffer.bufPtr()[i] != '\r')
            sbuffer.bufPtr()[parsep_len++] = sbuffer.bufPtr()[i];
    }
//    parsep[parsep_len] = '\0';

    char *linestart = nullptr;
    char *lineend = nullptr;
    char *keyp = nullptr;
    char *valuep = nullptr;
    linestart = strtok_r((char*) sbuffer.bufPtr(), "\n", &lineend);
    ret = parseHttpFirstLine(req, linestart);
    if (ret != HttpCode::HTTP_OK) {
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
            ret = HttpCode::HTTP_BAD_REQUEST;
            goto parse_end;
        }
    }

parse_end:
    return ret;
}

static int createResponse(HttpResponse &resp, SdsBytesBuf &sbuffer)
{
    std::string body = "";
    body.clear();
    body += "HTTP/1.1 " + std::to_string(resp.code) + "\r\n";
    for (auto it = resp.headers.begin(); it != resp.headers.end(); it++)
        body += it->first + ":" + it->second + "\r\n";

    body += "\r\n";
    sbuffer.writeBytes(body.c_str(), body.size());
    sbuffer.writeBytes(resp.buffer.bufPtr(), resp.buffer.size());
    return 0;
}

HttpServer::HttpServer()
    : detach(false), defaultContentType("text/html")
{
}

HttpServer::~HttpServer()
{
    this->shutdown();

    for (auto it = this->handlers.begin(); it != this->handlers.end(); it++)
        delete it->second;

    this->handlers.clear();
}


int HttpServer::handleConnection(int client_fd)
{
    int previous_error = 0;
    unsigned char buf[512];
    size_t nrecv = 0;
    size_t trecv = 0;
    SdsBytesBuf sbuffer;
    while ((nrecv = recv(client_fd, buf, sizeof(buf), 0))) {
        trecv += nrecv;
        sbuffer.writeBytes(buf, nrecv);
        if (nrecv < sizeof(buf))
            break;
    }
    sbuffer.writeUint8('\0');

    HttpRequest request;
    HttpResponse response;
    HttpCode ret = parseHttpReqest(request, sbuffer);
    if (ret == HttpCode::HTTP_OK) {
        this->handleRequest(request, response);
    } else {
        this->handleError(request, response, ret);
    }

    sbuffer.zero();
    createResponse(response, sbuffer);

    send(client_fd, sbuffer.bufPtr(), sbuffer.size(), 0);
    close(client_fd);

    return previous_error;
}

void *acceptFun(void *cls)
{
    HttpServer *srv = static_cast<HttpServer*>(cls);
    if (srv) {
        int client_fd;
        struct sockaddr_in address;
        socklen_t addrlen = sizeof(address);
        int clients_count = 0;
        pollfd wait_fds[MAX_POLL_FD_COUNT];
        wait_fds[0].fd = srv->server_fd;
        wait_fds[0].events = POLLIN | POLLPRI;
        while (srv->running) {
            if (poll(wait_fds, MAX_POLL_FD_COUNT, 3000) > 0) {
                int i;
                if ((wait_fds[0].revents & POLLIN)) {
                    if (clients_count >= MAX_CLIENT_COUNT) {
                        continue;
                    }

                    client_fd = accept(srv->server_fd, (struct sockaddr*) &address, &addrlen);
                    for (i = 1; i <= MAX_POLL_FD_COUNT; i++) {
                        if (wait_fds[i].fd == 0) {
                            clients_count++;
                            wait_fds[i].fd = client_fd;
                            wait_fds[i].events = POLLIN | POLLPRI;
                            break;
                        }
                    }
                }

                for (i = 1; i <= MAX_POLL_FD_COUNT; i++) {
                    client_fd = wait_fds[i].fd;
                    short int revents = wait_fds[i].revents;
                    if (client_fd > 0 && revents > 0) {
                        if ((revents & POLLHUP) || (revents & POLLERR)) {
                            close(wait_fds[i].fd);
                        } else if (revents & POLLIN) {
                            srv->handleConnection(client_fd);
                        }

                        wait_fds[i].fd = 0;
                        wait_fds[i].revents = 0;
                        clients_count--;
                    }
                }
            }
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
    this->running = true;

    if (detach) {
        pthread_create(&this->acceptThread, nullptr, &acceptFun, this);
    } else {
        acceptFun(this);
    }

    return 0;
}

int HttpServer::shutdown()
{
    this->running = false;
    pthread_join(this->acceptThread, nullptr);
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
    HttpCode code = HttpCode::HTTP_OK;

    response.headers["Content-Type"] = this->defaultContentType;
    if (this->hasHandlerFor(request.url)) {
        code = this->handlers[request.url]->handleRequest(request, response);
    } else {
        code = this->handleError(request, response, HttpCode::HTTP_NOT_FOUND);
    }

    return code;
}

HttpCode HttpServer::handleError(HttpRequest &request, HttpResponse &response, HttpCode errorCode)
{
    std::string ss;
    ss += "<html><body>";
    ss += "<h2> Error " + std::to_string(errorCode) + "</h2>";
    switch (errorCode) {
    case HttpCode::HTTP_NOT_FOUND:
        ss += "<h4>" + request.url + " Not Found" + "</h4>";
        break;
    case HttpCode::HTTP_INTERNAL_ERROR:
        ss += "<h4> Internal Server Error </h4>";
        break;
    case HttpCode::HTTP_NOT_IMPLEMENTED:
        ss += "<h4> Method Not Implemented </h4>";
        break;
    default:
        break;
    }
    ss + "</body></html>";
    response.buffer.writeBytes((unsigned char*) ss.c_str(), ss.size());
    return errorCode;
}
