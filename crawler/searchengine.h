#ifndef SEARCHENGINE_H
#define SEARCHENGINE_H

#include "sds_core/searchentriesdb.h"

struct SearchEngineConfigs {
    char *name;
    char *userAgent;
    char *searchQueryUrl;
    char *resultsContainerElement;
    char *resultContainerElement;
    char *resultUrlElement;
    char *resultUrlProperty;
    int resultUrlIsJson;
    char *resultUrlJsonProperty;
    char *resultTitleElement;
    char *resultTitleProperty;
    char *providedDataType;
};


class SearchEngine
{
public:
    SearchEngine(SearchEngineConfigs &configs_);

    void doSearch(std::vector<SearchEntry> &entries, const char *query);

private:
    int extractSearchResults(std::vector<SearchEntry> &entries, const char *url);

    SearchEngineConfigs configs;
};

#endif // SEARCHENGINE_H
