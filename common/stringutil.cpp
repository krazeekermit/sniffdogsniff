#include <string>
#include <cstring>
#include <vector>
#include <sstream>

#include "stringutil.h"

std::vector<std::string> split(const std::string &str, const char *delim)
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

std::string trim(const std::string &target, const char *cutset)
{

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
   return target.substr(startpos, endpos-startpos + 1);
}

std::string replace(std::string targetString, std::string oldValue, std::string newValue)
{
    size_t pos = targetString.find( oldValue );
    if( pos == std::string::npos ) {
        return targetString;
    }
    return targetString.replace(pos, oldValue.length(), newValue);
}

std::string replaceGlobal(std::string targetString, std::string oldValue, std::string newValue)
{
    int pos = 0;
    std::string resultString = "";
    size_t targetPos = targetString.find( oldValue, pos );
    while (targetPos != std::string::npos ) {
        std::string preOld = targetString.substr( pos, targetPos - pos );
        resultString += preOld + newValue;
        pos = targetPos + oldValue.length();
        targetPos = targetString.find( oldValue, pos );
    }
    resultString += targetString.substr(pos);
    return resultString;
}

std::string toLower(std::string in) {
    for (size_t i = 0; i < in.length(); i++) {
        in[i] = tolower(in[i]);
    }
    return in;
}

std::vector<std::string> tokenize(const std::string &str, const char *delimsset, const char *cutset)
{
    std::vector<std::string> tokens;
    std::string lowerStr = toLower(str);
    long begin = -1;
    size_t i;
    for (i = 0; i < lowerStr.length(); i++) {
        if (strchr(delimsset, lowerStr[i])) {
            tokens.push_back(trim(lowerStr.substr(begin, i - begin), cutset));
            begin = -1;
        } else if (begin < 0) {
            begin = i;
        }
    }
    tokens.push_back(trim(lowerStr.substr(begin, i - begin), cutset));
    return tokens;
}
