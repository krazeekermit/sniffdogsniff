#include "searchengine.h"

#include "crawlerutils.h"
#include "common/loguru.hpp"

SearchEngine::SearchEngine(SdsConfigFile::Section *section)
    :   name(section->lookupString("name")),
        userAgent(section->lookupString("user_agent")),
        searchQueryUrl(section->lookupString("search_query_url")),
        resultsContainerElement(section->lookupString("results_container_element")),
        resultContainerElement(section->lookupString("result_container_element")),
        resultUrlElement(section->lookupString("result_url_element")),
        resultUrlProperty(section->lookupString("result_url_property")),
        resultUrlIsJson(section->lookupBool("result_url_is_json")),
        resultUrlJsonProperty(section->lookupString("result_url_json_property")),
        resultTitleElement(section->lookupString("result_title_element")),
        resultTitleProperty(section->lookupString("result_title_property")),
        providedDataType(section->lookupString("provided_data_type"))
{}

void SearchEngine::doSearch(std::vector<SearchEntry> &entries, const char *query)
{
    LOG_F(1, "do search - %s", this->name.c_str());
    char searchUrl[512];
    sprintf(searchUrl, this->searchQueryUrl.c_str(), query);
    extractSearchResults(entries, searchUrl);
}

int SearchEngine::extractSearchResults(std::vector<SearchEntry> &entries, const char *url)
{
    GumboOutput *output = downloadWebDucument(url, this->userAgent.c_str());
    if (!output)
        return -1;

    GumboNode *docNode = getDocumentBody(output->root);
    if (!docNode)
        return -1;

    GumboNode *resultsContainerNode =
        getElementByAttr(docNode, attrb(this->resultsContainerElement), this->resultsContainerElement.c_str() + 1);
    if (resultsContainerNode) {
        int pos = 0;
        GumboNode *resultContainerNode = nullptr;
        GumboNode *resultUrlNode = nullptr;

        while ((resultContainerNode =
                getNextElementByAttr(resultsContainerNode, &pos, attrb(this->resultContainerElement), this->resultContainerElement.c_str() + 1))) {

            resultUrlNode = getElementByAttr(resultContainerNode, attrb(this->resultUrlElement), this->resultUrlElement.c_str() + 1);
            if (resultUrlNode) {
                std::string rlink;
                std::string title;
                if (this->resultUrlProperty.length() > 0) {
                    GumboAttribute *urlAttr = gumbo_get_attribute(&resultUrlNode->v.element.attributes, this->resultUrlProperty.c_str());
                    if (urlAttr)
                        rlink = urlAttr->value;
                } else {
                    getNodeText(resultUrlNode, rlink);
                }

                resultUrlNode = getElementByAttr(resultContainerNode, attrb(this->resultTitleElement), this->resultTitleElement.c_str() + 1);
                if (resultUrlNode) {
                    if (this->resultTitleProperty.length() > 0) {
                        GumboAttribute *titleAttr = gumbo_get_attribute(&resultUrlNode->v.element.attributes, this->resultTitleProperty.c_str());
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
