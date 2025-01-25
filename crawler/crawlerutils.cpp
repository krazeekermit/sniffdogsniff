#include "crawlerutils.h"

#include "common/logging.h"

#include <cstring>

GumboElementIterator::GumboElementIterator(GumboNode *parent, GumboTag tagType_)
    : GumboElementIterator(parent, nullptr, nullptr, tagType_)
{
}

GumboElementIterator::GumboElementIterator(GumboNode *parent, const char *attrName_, const char *attrValue_, GumboTag tagType_)
    : attrName(attrName_), attrValue(attrValue_), begin(nullptr), cur(nullptr), end(nullptr), tagType(tagType_)
{
    if (parent) {
        if (parent->type != GUMBO_NODE_ELEMENT)
            return;

        if (parent->v.element.tag == GUMBO_TAG_STYLE || parent->v.element.tag == GUMBO_TAG_SCRIPT ||  parent->v.element.tag == GUMBO_TAG_META)
            return;

        GumboVector *children = &parent->v.element.children;
        if (children->length) {
            this->begin = (GumboNode**) &children->data[0];
            this->cur = this->begin;
            this->end = (GumboNode**) &children->data[children->length];
        }
    }
}

GumboNode *GumboElementIterator::next()
{
    for (; this->cur < this->end; this->cur++) {
        GumboNode* pcur = *this->cur;
        if (pcur->type == GUMBO_NODE_ELEMENT && (this->tagType == GUMBO_TAG_LAST || pcur->v.element.tag == this->tagType)) {
            this->cur++;
            return pcur;
        }
    }
    return nullptr;
}

GumboNode *getNextElementByAttr(GumboNode *parent, int *pos, const char *attrName, const char *attrValue)
{
    GumboVector *children = &parent->v.element.children;
    if (children->length) {
        int i = pos ? *pos + 1 : 0;
        for (; i < children->length; i++) {
            GumboNode* child = (GumboNode*) children->data[i];
            if (child->type == GUMBO_NODE_ELEMENT) {
                GumboAttribute *attr = gumbo_get_attribute(&child->v.element.attributes, attrName);
                if (attr && strstr(attr->value, attrValue)) {
                    if (pos)
                        *pos = i;
                    return child;
                }
            }
        }
    }

    return nullptr;
}

GumboNode *getNextElement(GumboNode *parent, int *pos, GumboTag tagType)
{
    if (!parent)
        return nullptr;

    if (parent->type != GUMBO_NODE_DOCUMENT && parent->type != GUMBO_NODE_ELEMENT)
        return nullptr;

    if (parent->v.element.tag == GUMBO_TAG_STYLE || parent->v.element.tag == GUMBO_TAG_SCRIPT ||  parent->v.element.tag == GUMBO_TAG_META)
        return nullptr;

    GumboVector *children = &parent->v.element.children;
    if (children->length) {
        int i = pos ? *pos + 1 : 0;
        for (; i < children->length; i++) {
            GumboNode* child = (GumboNode*) children->data[i];
            if (child->type == GUMBO_NODE_ELEMENT && (tagType == GUMBO_TAG_LAST || child->v.element.tag == tagType)) {
                if (pos)
                    *pos = i;
                return child;
            }
        }
    }

    return nullptr;
}

GumboNode *getElementByAttr(GumboNode *parent, const char *attrName, const char *attrValue)
{
    if (!parent)
        return nullptr;

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

//GumboNode *getElement(GumboNode *parent, const char *attrName, const char *attrValue)
//{
//    if (!parent)
//        return nullptr;

//    if (parent->type != GUMBO_NODE_DOCUMENT && parent->type != GUMBO_NODE_ELEMENT)
//        return nullptr;

//    if (parent->v.element.tag == GUMBO_TAG_STYLE || parent->v.element.tag == GUMBO_TAG_SCRIPT ||  parent->v.element.tag == GUMBO_TAG_META)
//        return nullptr;

//    GumboAttribute *attr = gumbo_get_attribute(&parent->v.element.attributes, attrName);
//    if (attr && strstr(attr->value, attrValue)) {
//        return parent;
//    }

//    GumboVector *children = &parent->v.element.children;
//    if (children->length) {
//        int i;
//        for (i = 0; i < children->length; i++) {
//            GumboNode *e = (GumboNode*) children->data[i];
//            GumboNode *child = getElement(e, attrName, attrValue);
//            if (child) {
//                return child;
//            }
//        }
//    }

//    return nullptr;
//}

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

void getNodeText(GumboNode *parent, std::string &text)
{
    GumboVector *children = &parent->v.element.children;
    if (children->length == 1) {
        GumboNode *e = (GumboNode*) children->data[0];
        if (e->type == GUMBO_NODE_TEXT) {
            text = e->v.text.text;
        }
    }
}

/* curl write callback */
size_t writeCallback(char *in, uint size, uint nmemb, std::string *out)
{
  size_t r = size * nmemb;
  out->append(in, in+r);

  return r;
}

GumboOutput *downloadWebDucument(const char *url, const char *userAgent)
{
    std::string html_buf;

    CURL *curl;
//    char curl_errbuf[CURL_ERROR_SIZE];
    int err;

    curl = curl_easy_init();
    curl_easy_setopt(curl, CURLOPT_URL, url);
    curl_easy_setopt(curl, CURLOPT_USERAGENT, userAgent);
//    curl_easy_setopt(curl, CURLOPT_ERRORBUFFER, curl_errbuf);
    curl_easy_setopt(curl, CURLOPT_NOPROGRESS, 1L);
    curl_easy_setopt(curl, CURLOPT_VERBOSE, 0L);
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, writeCallback);

    curl_easy_setopt(curl, CURLOPT_WRITEDATA, &html_buf);
    err = curl_easy_perform(curl);
    if(err) {
        curl_easy_cleanup(curl);
        return nullptr;
    }

    /* clean-up */
    curl_easy_cleanup(curl);
    //return err;

    FILE *fp = fopen("./html_d.html", "w");
    fprintf(fp, "%s", html_buf.c_str());
    fclose(fp);

    GumboOutput *output = gumbo_parse_with_options(&kGumboDefaultOptions, html_buf.data(), html_buf.length());
    if (output->errors.length) {
        gumbo_destroy_output(&kGumboDefaultOptions, output);
        return nullptr;
    }

    logdebug << "parse ok for __ " << url;

    return output;
}
