#ifndef SEARCHENGINE_H
#define SEARCHENGINE_H

#include "sds_core/searchentriesdb.h"
#include "sds_core/sdsconfigfile.h"

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
    SearchEngine(SdsConfigFile::Section *section);


    void doSearch(std::vector<SearchEntry> &entries, const char *query);

private:
    int extractSearchResults(std::vector<SearchEntry> &entries, const char *url);

    std::string name;
    std::string userAgent;
    std::string searchQueryUrl;
    std::string resultsContainerElement;
    std::string resultContainerElement;
    std::string resultUrlElement;
    std::string resultUrlProperty;
    bool resultUrlIsJson;
    std::string resultUrlJsonProperty;
    std::string resultTitleElement;
    std::string resultTitleProperty;
    std::string providedDataType;
};

#endif // SEARCHENGINE_H
