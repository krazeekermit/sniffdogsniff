#include "netutil.h"

#include <errno.h>
#include <fcntl.h>
#include <sys/socket.h>
#include <arpa/inet.h>

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

    if (fcntl(fd, F_SETFL, opt | O_NONBLOCK) < 0) {
        return -1;
    }

    if (inet_pton(AF_INET, addr, &address.sin_addr) <= 0) {
        return -1;
    }

    address.sin_family = AF_INET;
    address.sin_port = htons(port);

    struct timeval timeout_val;
    timeout_val.tv_sec = timeout;
    timeout_val.tv_usec = 0;

    fd_set fd_wait;
    if (connect(fd, (struct sockaddr*)&address, sizeof(address)) < 0) {
        FD_ZERO(&fd_wait);
        FD_SET(fd, &fd_wait);

        err = select(fd + 1, NULL, &fd_wait, NULL, &timeout_val);
        if (err == 0) {
            errno = ETIMEDOUT;
            return -1;
        } else {
            addrlen = sizeof(err);
            if (getsockopt (fd, SOL_SOCKET, SO_ERROR, &err, &addrlen) < 0) {
                return -1;
            }

            if (err) {
                errno = err;
            }
        }
        return -1;
    }

    if (fcntl(fd, F_SETFL, opt) < 0) {
        return -1;
    }

    return fd;
}
