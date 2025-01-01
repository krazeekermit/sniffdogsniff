#ifndef NETUTIL_H
#define NETUTIL_H


#ifdef __cplusplus
extern "C" {
#endif

#include <string.h>
#include <stdio.h>
#include <stdlib.h>

inline int net_urlparse(char *addr, char *suffix, int *port, const char *url)
{
    const char *urlpp = strrchr(url, ':');
    if (urlpp)
        strncpy(addr, url, urlpp - url);
    else
        return -1;

    if (port)
        *port = atoi(urlpp);

    const char *supp = strrchr(url, '.');
    if (supp && suffix)
        strcpy(suffix, supp);

    return 0;
}

#ifdef __cplusplus
}
#endif

#endif // NETUTIL_H
