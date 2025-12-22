#include <string>
#include <cstring>
#include <vector>
#include <sstream>

#include "stringutil.h"

std::vector<std::string> StringUtil::split(const std::string &str, const char *delim)
{
    std::vector<std::string> splitstring;

    size_t seplen = strlen(delim);
    size_t begin = 0;
    size_t end = 0;
    while ((end = str.find(delim, begin)) != std::string::npos && begin <= str.length()) {
        splitstring.push_back(str.substr(begin, end - begin));
        begin = end+seplen;
    }
    splitstring.push_back(str.substr(begin, end - begin));

    return splitstring;
}

std::string StringUtil::trim(const std::string &target, const char *cutset)
{
    if (strlen(cutset) == 0) {
        return target;
    }

    size_t origlen = target.size();
    size_t startpos = -1;
    for(size_t i = 0; i < origlen; i++) {
        if(!strchr(cutset, target[i])) {
            startpos = i;
            break;
        }
    }

    size_t endpos = -1;
    for(size_t i = origlen - 1; i >= 0; i--) {
        if(!strchr(cutset, target[i])) {
            endpos = i;
            break;
        }
    }

    if(startpos == -1 || endpos == -1) {
        return "";
    }

    return target.substr(startpos, endpos - startpos + 1);
}

std::string StringUtil::toLower(std::string in) {
    for (size_t i = 0; i < in.length(); i++) {
        in[i] = tolower(in[i]);
    }
    return in;
}

std::vector<std::string> StringUtil::tokenize(const std::string &str, const char *delimsset, const char *cutset)
{
    std::vector<std::string> tokens;
    std::string lowerStr = toLower(str);
    size_t i = 0;
    while (i < lowerStr.size()) {
        while (strchr(delimsset, lowerStr[i]) && i < lowerStr.size()) {
            i++;
        }

        if (i >= lowerStr.size()) {
            break;
        }

        std::string tok = "";
        while (strchr(delimsset, lowerStr[i]) == nullptr && i < lowerStr.size()) {
            tok += lowerStr[i];
            i++;
        }
        tokens.push_back(trim(tok, cutset));
    }

    return tokens;
}
