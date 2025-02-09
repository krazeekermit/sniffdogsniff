#ifndef HTTPSERVER_H
#define HTTPSERVER_H

#include "common/sdsbytesbuf.h"

#include <pthread.h>
#include <iostream>
#include <sstream>
#include <map>
#include <functional>
#include <deque>

enum HttpCode {
    // Http success codes
    HTTP_OK = 200,
    HTTP_CREATED = 201,
    HTTP_ACCEPTED = 202,
    HTTP_NO_CONTENT = 204,
    HTTP_RESET_CONTENT = 205,
    HTTP_PARTIAL_CONTENT = 206,

    // Http error codes
    HTTP_BAD_REQUEST = 400,
    HTTP_UNAUTHORIZED = 401,
    HTTP_FORBIDDEN = 403,
    HTTP_NOT_FOUND = 404,
    HTTP_METHOD_NOT_ALLOWED = 405,
    HTTP_NOT_ACCEPTABLE = 406,
    HTTP_REQUEST_TIMEOUT = 408,

    // Http internal error codes
    HTTP_INTERNAL_ERROR = 500,
    HTTP_NOT_IMPLEMENTED = 501
};

enum HttpMethod {
    HTTP_GET,
    HTTP_HEAD,
    HTTP_POST
};

struct HttpRequest {
    std::string url;
    HttpMethod method;
    std::string version;
    std::map<std::string, std::string> headers;
    std::map<std::string, std::string> values;
};

struct HttpResponse {
    std::map<std::string, std::string> headers;
    SdsBytesBuf buffer;
    HttpCode code;

    void writeResponse(std::string str)
    {
        this->buffer.writeBytes(str.c_str(), str.length());
    }

    void writeResponse(const uint8_t *buffer, size_t bufferSize)
    {
        this->buffer.writeBytes(buffer, bufferSize);
    }
};

class HttpRequestHandler {
public:
    virtual HttpCode handleRequest(HttpRequest &request, HttpResponse &response)
    {
        return HttpCode::HTTP_OK;
    }

};

class HttpServer
{
public:
    HttpServer();
    ~HttpServer();

    int startListening(const char *addr, int port, bool detach = false);
    int shutdown();

    void addHandler(std::string u, HttpRequestHandler *h);
    void removeHandler(std::string u);

protected:
    bool hasHandlerFor(std::string u);

    virtual HttpCode handleRequest(HttpRequest &request, HttpResponse &response);
    virtual HttpCode handleError(HttpRequest &request, HttpResponse &response, HttpCode errorCode);

private:
    bool detach;
    bool running;
    pthread_t acceptThread;
    std::map<std::string, HttpRequestHandler*> handlers;
    int server_fd;
    std::string defaultContentType;

    int handleConnection(int client_fd);

    friend void *acceptFun(void *cls);
};

#endif // HTTPSERVER_H
