#include "searchengine.h"

#include "crawlerutils.h"
#include "common/loguru.hpp"

SearchEngine::SearchEngine(SearchEngineConfigs &configs_)
    : configs(configs_)
{}

void SearchEngine::doSearch(std::vector<SearchEntry> &entries, const char *query)
{
    LOG_F(1, "do search - ", this->configs.name);
    char searchUrl[512];
    sprintf(searchUrl, this->configs.searchQueryUrl, query);
    extractSearchResults(entries, searchUrl);
}

int SearchEngine::extractSearchResults(std::vector<SearchEntry> &entries, const char *url)
{
    GumboOutput *output = downloadWebDucument(url, this->configs.userAgent);
    if (!output)
        return -1;

    GumboNode *docNode = getDocumentBody(output->root);
    if (!docNode)
        return -1;

    GumboNode *resultsContainerNode =
            getElementByAttr(docNode, attrb(this->configs.resultsContainerElement), this->configs.resultsContainerElement + 1);
    if (resultsContainerNode) {
        int pos = 0;
        GumboNode *resultContainerNode = nullptr;
        GumboNode *resultUrlNode = nullptr;

        while ((resultContainerNode =
               getNextElementByAttr(resultsContainerNode, &pos, attrb(this->configs.resultContainerElement), this->configs.resultContainerElement + 1))) {

            resultUrlNode = getElementByAttr(resultContainerNode, attrb(this->configs.resultUrlElement), this->configs.resultUrlElement + 1);
            if (resultUrlNode) {
                std::string rlink;
                std::string title;
                if (this->configs.resultUrlProperty) {
                    GumboAttribute *urlAttr = gumbo_get_attribute(&resultUrlNode->v.element.attributes, this->configs.resultUrlProperty);
                    if (urlAttr)
                        rlink = urlAttr->value;
                } else {
                    getNodeText(resultUrlNode, rlink);
                }

                resultUrlNode = getElementByAttr(resultContainerNode, attrb(this->configs.resultTitleElement), this->configs.resultTitleElement + 1);
                if (resultUrlNode) {
                    if (this->configs.resultTitleProperty) {
                        GumboAttribute *titleAttr = gumbo_get_attribute(&resultUrlNode->v.element.attributes, this->configs.resultTitleProperty);
                        if (titleAttr)
                            title = titleAttr->value;
                    } else {
                        getNodeText(resultUrlNode, title);
                    }
                }

                if (title.size() > 0) {
                    entries.emplace_back(title, rlink, SearchEntry::Type::SITE);
                }
            }
        }
    }

    gumbo_destroy_output(&kGumboDefaultOptions, output);
    return 0;
}
