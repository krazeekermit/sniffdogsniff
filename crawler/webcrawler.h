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
    ~WebCrawler();

    int load(const char *path);
    int save(const char *path);

    void startCrawling();
    void stopCrawling();
    int getEntriesForBroadcast(std::vector<SearchEntry> &entries);

    void doSearch(std::vector<SearchEntry> &entries, const char *query);

    static void *crawlingFunc(void *p);

private:
    bool run;
    pthread_t thread;
    pthread_mutex_t mutex;
    pthread_cond_t cond;
    std::deque<std::string> urlQueue;
    std::vector<SearchEntry> searches;

    std::vector<SearchEngine> searchEngines;
};

#endif // WEBCRAWLER_H
