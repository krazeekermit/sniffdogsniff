#ifndef CRAWLERUTILS_H
#define CRAWLERUTILS_H

#include <gumbo.h>
#include <curl/curl.h>

#include <iostream>

#define attrb(S) S[0] == '#' ? "id" : "class"

class GumboElementIterator
{
public:
    GumboElementIterator(GumboNode *parent, GumboTag tagType_ = GumboTag::GUMBO_TAG_LAST);
    GumboElementIterator(GumboNode *parent, const char *attrName_, const char *attrValue_, GumboTag tagType_ = GumboTag::GUMBO_TAG_LAST);

    GumboNode *next();

private:
    GumboTag tagType;
    const char *attrName;
    const char *attrValue;

    GumboNode **begin;
    GumboNode **cur;
    GumboNode **end;
};

GumboNode *getNextElementByAttr(GumboNode *parent, int *pos, const char *attrName, const char *attrValue);
GumboNode *getNextElement(GumboNode *parent, int *pos, GumboTag tagType);
GumboNode *getElementByAttr(GumboNode *parent, const char *attrName, const char *attrValue);
GumboNode *getDocumentBody(GumboNode *parent);
void getNodeText(GumboNode *parent, std::string &text);

GumboOutput *downloadWebDucument(const char *url, const char *userAgent);


#endif // CRAWLERUTILS_H
