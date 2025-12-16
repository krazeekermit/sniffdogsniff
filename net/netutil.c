#include "netutil.h"

#include <errno.h>
#include <fcntl.h>
#include <poll.h>
#include <unistd.h>
#include <sys/socket.h>
#include <arpa/inet.h>
#include <netdb.h>


size_t bytes_to_hex_string(char *out, const unsigned char *in, size_t in_sz)
{
    size_t i;
    size_t h_sz = 0;
    for (i = 0; i < in_sz; i++) {
        unsigned char hi = (in[i] >> 4) & 0x0f;
        unsigned char lo = (in[i] & 0x0f);
        out[h_sz++] = (hi > 9 ? ('A' + (hi - 10)) : ('0' + hi));
        out[h_sz++] = (lo > 9 ? ('A' + (lo - 10)) : ('0' + lo));
    }
    return out[h_sz] = '\0';
}

size_t hex_string_to_bytes(unsigned char *out, const char *in)
{
    size_t i = 0;
    size_t h_sz = 0;
    while (i < strlen(in)) {
        char hi = in[i++];
        char lo = in[i++];
        if (hi >= '0' && hi <= '9')
            hi = hi - '0';
        else if (hi >= 'A' && hi <= 'F')
            hi = hi - 'A' + 10;

        if (lo >= '0' && lo <= '9')
            lo = lo - '0';
        else if (lo >= 'A' && lo <= 'F')
            lo = lo - 'A' + 10;
        out[h_sz++] = ((hi << 4) & 0xf0) | (lo & 0x0f);
    }
    return h_sz;
}

int net_socket_connect(const char *addr, int port, long timeout)
{
    int fd, opt;
    int err = 0;
    ssize_t valread;
    struct sockaddr_in address;
    socklen_t addrlen = sizeof(address);

    fd = socket(AF_INET, SOCK_STREAM, 0);
    if (fd < 0)
        return -1;

    if ((opt = fcntl(fd, F_GETFL, NULL)) < 0) {
        return -1;
    }

    if (timeout > 0) {
        if (fcntl(fd, F_SETFL, opt | O_NONBLOCK) < 0) {
            return -1;
        }
    }

    if (inet_pton(AF_INET, addr, &address.sin_addr) <= 0) {
        struct hostent *host_ent = NULL;

        host_ent = gethostbyname(addr);
        if (!host_ent) {
            return -1;
        }
        int i = 0;
        if (host_ent->h_addr_list[0] != NULL) {
            memcpy(&address.sin_addr, host_ent->h_addr_list[0], sizeof(struct in_addr));
        }
    }

    address.sin_family = AF_INET;
    address.sin_port = htons(port);

    struct pollfd wait_fds[1];

    wait_fds[0].fd = fd;
    wait_fds[0].events = POLLIN | POLLPRI | POLLOUT;
    connect(fd, (struct sockaddr*)&address, sizeof(address));
    if (errno != EINPROGRESS) {
        return -1;
    }

    if (poll(wait_fds, 1, timeout) > 0) {
        if ((wait_fds[0].revents & POLLHUP) || (wait_fds[0].revents & POLLERR)) {
            addrlen = sizeof(err);
            if (getsockopt(fd, SOL_SOCKET, SO_ERROR, &err, &addrlen) < 0) {
                return -1;
            }

            if (err) {
                close(fd);
                errno = err;
                return -1;
            }
        } else {
            if (fcntl(fd, F_SETFL, opt) < 0) {
                return -1;
            }
            return fd;
        }
    }

    close(fd);
    errno = ETIMEDOUT;
    return -1;
}
