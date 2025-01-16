#ifndef WEBCRAWLER_H
#define WEBCRAWLER_H

#include "searchengine.h"

#include "sds_config.h"

#include <pthread.h>

#include <deque>

class WebCrawler
{
public:
    WebCrawler(SdsConfig &cfg);

    void startCrawling();
    void doSearch(std::vector<SearchEntry> &entries, const char *query);

    static void *crawlingFunc(void *p);

private:
    pthread_t thread;
    pthread_mutex_t mutex;
    pthread_cond_t cond;
    std::deque<std::string> urlQueue;
    std::vector<SearchEntry> searches;

    std::vector<SearchEngine> searchEngines;
};

#endif // WEBCRAWLER_H
