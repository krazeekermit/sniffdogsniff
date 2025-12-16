#ifndef NETUTIL_H
#define NETUTIL_H


#ifdef __cplusplus
extern "C" {
#endif

#include <string.h>
#include <stdio.h>
#include <stdlib.h>

size_t bytes_to_hex_string(char *out, const unsigned char *in, size_t in_sz);

size_t hex_string_to_bytes(unsigned char *out, const char *in);

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
        *port = atoi(urlpp + 1);

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
