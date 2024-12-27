#ifndef SOCKS5_H
#define SOCKS5_H

#ifdef __cplusplus
extern "C" {
#endif

const char *socks5_strerror(int n);

int socks5_connect(const char *socks5_addr, int socks5_port, const char *addr, int port);

#ifdef __cplusplus
}
#endif

#endif // SOCKS5_H
