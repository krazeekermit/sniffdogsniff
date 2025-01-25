#include "webcrawler.h"

#include "crawlerutils.h"
#include "common/logging.h"

#include <map>
#include <cstring>

WebCrawler::WebCrawler(SdsConfig &cfg)
{
    pthread_mutex_init(&this->mutex, nullptr);
    pthread_cond_init(&this->cond, nullptr);

    for (int i = 0; i < cfg.search_engines.size(); i++) {
        this->searchEngines.emplace_back(cfg.search_engines[i]);
    }
}

WebCrawler::~WebCrawler()
{
    this->stopCrawling();
    pthread_mutex_destroy(&this->mutex);
    pthread_cond_destroy(&this->cond);
}

int WebCrawler::load(const char *path)
{
    FILE *fp = fopen(path, "r");
    if (!fp)
        return -1;

    pthread_mutex_lock(&this->mutex);
    char buffer[1024];
    while (fgets(buffer, sizeof(buffer), fp)) {
        char *endp = strchr(buffer, '\n');
        if (endp)
            *endp = '\0';
        this->urlQueue.emplace_back(endp);
    }
    fclose(fp);
    pthread_cond_signal(&this->cond);
    pthread_mutex_unlock(&this->mutex);

    return 0;
}

int WebCrawler::save(const char *path)
{
    FILE *fp = fopen(path, "w");
    if (!fp)
        return -1;

    pthread_mutex_lock(&this->mutex);
    for (int i = 0; i < this->urlQueue.size(); i++) {
        fprintf(fp, "%s\n", this->urlQueue[i].c_str());
    }
    fclose(fp);
    pthread_cond_signal(&this->cond);
    pthread_mutex_unlock(&this->mutex);

    return 0;
}

static void scanDocument(GumboNode *parent, std::string siteUrl, std::map<std::string, SearchEntry> &entries)
{
    std::string rlink = "";
    std::string title = "";
    GumboNode *node = nullptr;
    GumboElementIterator iter(parent);
    while ((node = iter.next())) {
        if (node->v.element.tag == GumboTag::GUMBO_TAG_A) {
            GumboAttribute *attr = gumbo_get_attribute(&node->v.element.attributes, "href");
            if (attr) {
                getNodeText(node, title);
                rlink = attr->value;

                if (rlink.find("://") == std::string::npos) {
                    rlink = siteUrl + rlink;
                }

                SearchEntry se(title, rlink, SearchEntryType::SITE);
                entries[rlink] = se;
                logdebug << "crawler found " << se;
            }
        } else {
            scanDocument(node, siteUrl, entries);
        }
    }
}

void *WebCrawler::crawlingFunc(void *p)
{
    WebCrawler *crawler = static_cast<WebCrawler*>(p);
    if (crawler) {
        std::map<std::string, SearchEntry> entries;
        while (crawler->run) {
            pthread_mutex_lock(&crawler->mutex);
            while (crawler->urlQueue.empty()) {
                if (!crawler->run) {
                    goto crawling_end;
                }

                pthread_cond_wait(&crawler->cond, &crawler->mutex);
            }
            std::string url = crawler->urlQueue.front();
            crawler->urlQueue.pop_front();
            pthread_mutex_unlock(&crawler->mutex);

            std::string userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.0.0 Safari/537.36";
            GumboOutput *output = downloadWebDucument(url.c_str(), userAgent.c_str());

            if (output) {
                GumboNode *body = getDocumentBody(output->root);
                if (body) {
                    entries.clear();
                    scanDocument(body, url, entries);

                    pthread_mutex_lock(&crawler->mutex);
                    for (auto it = entries.begin(); it != entries.end(); it++) {
                        crawler->urlQueue.push_back(it->first);
                        crawler->searches.push_back(it->second);
                    }
                    logdebug << "crawler found " << entries.size() << "new results";
                    pthread_mutex_unlock(&crawler->mutex);
                }
            }
        }
    }
crawling_end:

    logdebug << "crawler shutted down!";
    return nullptr;
}

void WebCrawler::startCrawling()
{
    pthread_create(&this->thread, nullptr, &crawlingFunc, this);
    this->run = true;
}

void WebCrawler::stopCrawling()
{
    pthread_mutex_lock(&this->mutex);
    this->run = false;
    pthread_cond_signal(&this->cond);
    pthread_mutex_unlock(&this->mutex);

    void *dummy = nullptr;
    pthread_join(this->thread, &dummy);
}

int WebCrawler::getEntriesForBroadcast(std::vector<SearchEntry> &entries)
{
    int finds = 0;
    pthread_mutex_lock(&this->mutex);
    for (auto it = this->searches.begin(); it != this->searches.end(); it++) {
        entries.push_back(*it);
    }
    finds = this->searches.size();
    this->searches.clear();
    pthread_cond_signal(&this->cond);
    pthread_mutex_unlock(&this->mutex);

    return finds;
}

void WebCrawler::doSearch(std::vector<SearchEntry> &entries, const char *query)
{
    for (auto it = this->searchEngines.begin(); it != this->searchEngines.end(); it++) {
        it->doSearch(entries, query);
    }

    pthread_mutex_lock(&this->mutex);
    for (auto it = entries.begin(); it != entries.end(); it++) {
        this->urlQueue.push_back(it->getUrl());
    }
    pthread_cond_signal(&this->cond);
    pthread_mutex_unlock(&this->mutex);

    logdebug << "Crawler seeded with " << entries.size() << " new urls";
}
