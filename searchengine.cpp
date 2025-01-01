#include "searchengine.h"

#include "logging.hpp"
#include <gumbo.h>


#include <curl/curl.h>
#include <locale>
#include <codecvt>

SearchEngine::SearchEngine(SearchEngineConfigs &configs_)
    : configs(configs_)
{}

void SearchEngine::doSearch(std::vector<SearchEntry> &entries, const char *query)
{
    char searchUrl[512];
    sprintf(searchUrl, this->configs.searchQueryUrl, query);
    extractSearchResults(entries, searchUrl);
}

/* curl write callback */
size_t writeCallback(char *in, uint size, uint nmemb, std::string *out)
{
  size_t r = size * nmemb;
  out->append(in, in+r);

  return r;
}

#define attrb(S) S[0] == '#' ? "id" : "class"
static GumboNode *getNextElementByAttr(GumboNode *parent, int *pos, const char *attrName, const char *attrValue)
{
    GumboVector *children = &parent->v.element.children;
    if (children->length) {
        int i = pos ? *pos + 1 : 0;
        for (; i < children->length; i++) {
            GumboNode* child = (GumboNode*) children->data[i];
            GumboAttribute *attr = gumbo_get_attribute(&child->v.element.attributes, attrName);
            if (attr && strstr(attr->value, attrValue)) {
                if (pos)
                    *pos = i;
                return child;
            }
        }
    }

    return nullptr;
}

static GumboNode *getElementByAttr(GumboNode *parent, const char *attrName, const char *attrValue)
{
    if (!parent)
        return nullptr;


//    const char *tname = parent->v.element.tag == GUMBO_TAG_UNKNOWN ? "???" : gumbo_normalized_tagname(parent->v.element.tag);
//        logdebug(<< "node type " << parent->type << ", "<< tname
//                 << ", childrens " << parent->v.element.children.length);

    if (parent->type != GUMBO_NODE_DOCUMENT && parent->type != GUMBO_NODE_ELEMENT)
        return nullptr;

    if (parent->v.element.tag == GUMBO_TAG_STYLE || parent->v.element.tag == GUMBO_TAG_SCRIPT ||  parent->v.element.tag == GUMBO_TAG_META)
        return nullptr;

    GumboAttribute *attr = gumbo_get_attribute(&parent->v.element.attributes, attrName);
    if (attr && strstr(attr->value, attrValue)) {
        return parent;
    }

    GumboVector *children = &parent->v.element.children;
    if (children->length) {
        int i;
        for (i = 0; i < children->length; i++) {
            GumboNode *e = (GumboNode*) children->data[i];
            GumboNode *child = getElementByAttr(e, attrName, attrValue);
            if (child) {
                return child;
            }
        }
    }

    return nullptr;
}

GumboNode *getDocumentBody(GumboNode *parent) {
    if (!parent)
        return nullptr;

    GumboVector *children = &parent->v.element.children;
    if (children->length) {
        int i;
        for (i = 0; i < children->length; i++) {
            GumboNode *e = (GumboNode*) children->data[i];
            if (e && e->v.element.tag == GUMBO_TAG_BODY) {
                return e;
            }
        }
    }
    return nullptr;
}

static void getNodeText(GumboNode *parent, std::string &text)
{
    GumboVector *children = &parent->v.element.children;
    if (children->length == 1) {
        GumboNode *e = (GumboNode*) children->data[0];
        if (e->type == GUMBO_NODE_TEXT) {
            text = e->v.text.text;
        }
    }
}

int SearchEngine::extractSearchResults(std::vector<SearchEntry> &entries, const char *url)
{
    std::string html_buf;

    CURL *curl;
    char curl_errbuf[CURL_ERROR_SIZE];
    int err;

    curl = curl_easy_init();
    curl_easy_setopt(curl, CURLOPT_URL, url);
    curl_easy_setopt(curl, CURLOPT_USERAGENT, this->configs.userAgent);
    curl_easy_setopt(curl, CURLOPT_ERRORBUFFER, curl_errbuf);
    curl_easy_setopt(curl, CURLOPT_NOPROGRESS, 1L);
    curl_easy_setopt(curl, CURLOPT_VERBOSE, 0L);
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, writeCallback);

    curl_easy_setopt(curl, CURLOPT_WRITEDATA, &html_buf);
    err = curl_easy_perform(curl);
    if(err) {
        logerr << "error opening " << url << ": " << curl_errbuf;
        curl_easy_cleanup(curl);
        return err;
    }

    /* clean-up */
    curl_easy_cleanup(curl);
    //return err;

    GumboOutput *output = gumbo_parse_with_options(&kGumboDefaultOptions, html_buf.data(), html_buf.length());
    if (output->errors.length) {
        int i;
        for (i = 0; i < output->errors.length; i++)
            logdebug << "error opening %s: parser collected errors " << url << ": " << i;

        gumbo_destroy_output(&kGumboDefaultOptions, output);
        return -1;
    }
    GumboNode *doc = getDocumentBody(output->root);

    GumboNode *resultsContainerNode =
            getElementByAttr(doc, attrb(this->configs.resultsContainerElement), this->configs.resultsContainerElement + 1);
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

                entries.emplace_back(title, rlink, SearchEntryType::SITE);
            }
        }
    }

    gumbo_destroy_output(&kGumboDefaultOptions, output);
    return err;
}
