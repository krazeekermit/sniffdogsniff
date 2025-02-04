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
    if (urlpp) {
        size_t len = urlpp - url;
        strncpy(addr, url, len);
        addr[len] = '\0';
    }
    else
        return -1;

    if (port && urlpp)
        *port = atoi(urlpp);

    const char *supp = strrchr(addr, '.');
    if (supp && suffix) {
        strcpy(suffix, supp);
    }

    return 0;
}

int net_socket_connect(const char *addr, int port, long timeout);

#ifdef __cplusplus
}
#endif

#endif // NETUTIL_H
