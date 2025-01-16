#include "webcrawler.h"

#include "crawlerutils.h"
#include "logging.h"

#include <map>

WebCrawler::WebCrawler(SdsConfig &cfg)
{
    pthread_mutex_init(&this->mutex, nullptr);
    pthread_cond_init(&this->cond, nullptr);

    for (int i = 0; i < cfg.search_engines.size(); i++) {
        this->searchEngines.emplace_back(cfg.search_engines[i]);
    }
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
        while (1) {
            pthread_mutex_lock(&crawler->mutex);
            while (crawler->urlQueue.empty()) {
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
    return nullptr;
}

void WebCrawler::startCrawling()
{
    pthread_create(&this->thread, nullptr, &crawlingFunc, this);
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
