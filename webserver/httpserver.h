#ifndef HTTPSERVER_H
#define HTTPSERVER_H

#include <pthread.h>
#include <iostream>
#include <sstream>
#include <map>
#include <functional>
#include <deque>

enum HttpMethod {
    HTTP_GET, HTTP_POST
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
    std::string ss;
    int code;
};

class HttpRequestHandler {
public:
    virtual int handleRequest(HttpRequest &request, HttpResponse &response)
    {
        return 201;
    }

};

class HttpServer
{
public:
    HttpServer();
    ~HttpServer();

    int startListening(const char *addr, int port);
    int shutdown();

    void addHandler(std::string u, HttpRequestHandler *h);
    void removeHandler(std::string u);

protected:
    bool hasHandlerFor(std::string u);

    virtual int handleRequest(HttpRequest &request, HttpResponse &response);
    virtual int handleError(HttpRequest &request, HttpResponse &response, int errorCode);

private:
    pthread_t *threadPool;
    pthread_mutex_t mutex;
    pthread_cond_t cond;
    std::deque<int> clientsQueue;
    std::map<std::string, HttpRequestHandler*> handlers;
    int server_fd;
    std::string defaultContentType;

    friend void *accessHandlerCallback(void *cls);
};

#endif // HTTPSERVER_H
