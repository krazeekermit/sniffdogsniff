#include <sys/socket.h>
#include <arpa/inet.h>
#include <unistd.h>
#include <string.h>

const char *socks5_strerror(int n)
{
    switch (-n) {
    case 0x01:
        return "general SOCKS server failure";
    case 0x02:
        return "connection not allowed by ruleset";
    case 0x03:
        return "Network unreachable";
    case 0x04:
        return "Host unreachable";
    case 0x05:
        return "Connection refused";
    case 0x06:
        return "TTL expired";
    case 0x07:
        return "Command not supported";
    case 0x08:
        return "Address type not supported";
    default:
        return "unknown error";
    }
}

int socks5_connect(const char *socks5_addr, int socks5_port, const char *addr, int port)
{
    int i, fd;
    int opt = 0;
    size_t valread;
    struct sockaddr_in address;
    socklen_t addrlen = sizeof(address);

    unsigned char buf[512];

    fd = socket(AF_INET, SOCK_STREAM, 0);
    if (fd < 0)
        return -1;

    if (inet_pton(AF_INET, socks5_addr, &address.sin_addr) <= 0) {
        return -1;
    }

    address.sin_family = AF_INET;
    address.sin_port = htons(socks5_port);

    if (connect(fd, (struct sockaddr*)&address, sizeof(address)) < 0) {
        return -1;
    }

    // refs https://www.rfc-editor.org/rfc/rfc1928
    buf[0] = 0x05;
    buf[1] = 0x01;
    buf[2] = 0x00;

    if (send(fd, buf, 3, 0) != 3) {
        close(fd);
        return -1;
    }

    size_t buf_sz = 0;
    unsigned char address_sz;
    unsigned char addrType = 0x01;

    buf[buf_sz++] = 0x05; // version = 5
    buf[buf_sz++] = 0x01; // connect
    buf[buf_sz++] = 0x00; // reserved

    struct sockaddr_in baddr;
    if (inet_pton(AF_INET, addr, &baddr.sin_addr) > 0) {
        buf[buf_sz++] = addrType = 0x01; // addr type (ipv4)
    } else {
        buf[buf_sz++] = addrType = 0x03; // addr type (domain)
        buf[buf_sz++] = strlen(addr);
    }

    if (send(fd, buf, buf_sz, 0) != buf_sz) {
        close(fd);
        return -1;
    }

    if (addrType == 0x01) {
        if (send(fd, &baddr.sin_addr, sizeof(baddr.sin_addr), 0) != 1) {
            close(fd);
            return -1;
        }
    } else if (addrType == 0x03) {
        size_t addr_len = strlen(addr);
        if (send(fd, addr, sizeof(char) * addr_len, 0) != addr_len) {
            close(fd);
            return -1;
        }
    }

    uint16_t nsport = htons(port);
    if (send(fd, &nsport, sizeof(uint16_t), 0) != sizeof(uint16_t)) {
        close(fd);
        return -1;
    }

    memset(buf, 0, sizeof(buf));
    if (recv(fd, buf, 4, 0) != 4) {
        close(fd);
        return -1;
    }

    if (buf[0] != 0x05) {
        close(fd);
        return -1;
    }

    if (buf[1] != 0x00) {
        close(fd);
        return -buf[1];
    }

    if (buf[3] != addrType) {
        close(fd);
        return -1;
    }

    if (recv(fd, &address_sz, 1, 0) != 1) {
        close(fd);
        return -1;
    }
    if (recv(fd, buf, address_sz, 0) != address_sz) {
        close(fd);
        return -1;
    }

    if (recv(fd, &nsport, sizeof(uint16_t), 0) != sizeof(uint16_t)) {
        close(fd);
        return -1;
    }

    return fd;
}
