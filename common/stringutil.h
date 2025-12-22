#ifndef STRINGUTILS_H
#define STRINGUTILS_H

#include <vector>
#include <string>
#include <cstring>
#include <sstream>
#include <iostream>
#include <cstdlib>

namespace StringUtil
{

std::vector<std::string> split(const std::string &str, const char *delim);
std::string trim(const std::string &target, const char *cutset = " \r\n");

std::string toLower(std::string in);
std::vector<std::string> tokenize(const std::string &str, const char *delimsset = " \r\n", const char *cutset = "");

}

#endif // STRINGUTIL_H

