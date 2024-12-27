#ifndef UTILS_H
#define UTILS_H

#define UNIX_HOUR 3600
#define UNIX_DAY  86400

#include <iostream>
#include <string.h>

static inline int fgetstdstr(std::string &s, FILE *fp)
{
    s.clear();
    char c = '\0';
    while ((c = fgetc(fp)) != '\0')
        s += c;

    s += '\0';
    return s.length();
}

#endif // UTILS_H
