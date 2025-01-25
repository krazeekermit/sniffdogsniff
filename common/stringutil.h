#ifndef STRINGUTILS_H
#define STRINGUTILS_H

#include <vector>
#include <string>
#include <cstring>
#include <sstream>
#include <iostream>
#include <cstdlib>

#define strcpy_safe(D, S, L) strncpy(D, S, L);

class IHasToString {
public:
    virtual std::string toString() = 0;
};

template<typename T>
std::string toString(T val ) { // not terribly efficient, but works...
   std::ostringstream myostringstream;
   myostringstream << val;
   return myostringstream.str();
}

std::vector<std::string> split(const std::string &str, const char *delim);
std::string trim(const std::string &target, const char *cutset = " \r\n");

// returns empty string if off the end of the number of available tokens
inline std::string getToken( std::string targetstring, int tokenIndexFromZero, std::string separator = " " ) {
   std::vector<std::string> splitstring = split( targetstring, separator.c_str() );
   if( tokenIndexFromZero < (int)splitstring.size() ) {
      return splitstring[tokenIndexFromZero];
   } else {
      return "";
   }
}

std::string replace(std::string targetString, std::string oldValue, std::string newValue);
std::string replaceGlobal(std::string targetString, std::string oldValue, std::string newValue);

std::string toLower(std::string in);
std::vector<std::string> tokenize(const std::string &str, const char *delimsset = " ", const char *cutset = " \r\n");

#endif // STRINGUTIL_H

