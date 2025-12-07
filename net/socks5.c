#include <sys/socket.h>
#include <arpa/inet.h>
#include <unistd.h>
#include <string.h>

#define SOCKS_VERSION 0x05

#define CONNECT_CMD 0x01
#define BIND_CMD 0x02

#define METHOD_NO_AUTH_REQUIRED 0x00
#define NO_ACCEPTABLE_METHODS 0xff

#define ADDR_IP_V4 0x01
#define ADDR_DOMAIN 0x03
#define ADDR_IP_V6 0x04 // note: as of now ipv6 is not implemented

#define NO_ERROR 0x00

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
    int fd;
    struct sockaddr_in address;
    unsigned char buf[256];

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
    /*
     * +----+----------+----------+
     * |VER | NMETHODS | METHODS  |
     * +----+----------+----------+
     * | 1  |    1     | 1 to 255 |
     * +----+----------+----------+
     */

    buf[0] = SOCKS_VERSION;
    buf[1] = 0x01;
    buf[2] = METHOD_NO_AUTH_REQUIRED;

    if (send(fd, buf, 3, 0) != 3) {
        close(fd);
        return -1;
    }

    memset(buf, 0, sizeof(buf));
    if (recv(fd, buf, 2, 0) != 2) {
        close(fd);
        return -1;
    }

    if (buf[0] != SOCKS_VERSION || buf[1] == NO_ACCEPTABLE_METHODS) {
        close(fd);
        return -1;
    }

    /*
     * +----+-----+-------+------+----------+----------+
     * |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
     * +----+-----+-------+------+----------+----------+
     * | 1  |  1  | X'00' |  1   | Variable |    2     |
     * +----+-----+-------+------+----------+----------+
    */

    size_t buf_sz = 0;
    unsigned char addr_len = 0;
    unsigned char addr_type = ADDR_IP_V4;
    buf[buf_sz++] = SOCKS_VERSION; // version = 5
    buf[buf_sz++] = CONNECT_CMD; // connect
    buf[buf_sz++] = 0x00; // reserved

    struct sockaddr_in baddr;
    if (inet_pton(AF_INET, addr, &baddr.sin_addr) > 0) {
        buf[buf_sz++] = addr_type = ADDR_IP_V4; // addr type (ipv4)
    } else {
        buf[buf_sz++] = addr_type = ADDR_DOMAIN; // addr type (domain)
        buf[buf_sz++] = addr_len = strlen(addr);
    }

    if (send(fd, buf, buf_sz, 0) != buf_sz) {
        close(fd);
        return -1;
    }

    switch (addr_type) {
    case ADDR_IP_V4:
        if (send(fd, &baddr.sin_addr, sizeof(baddr.sin_addr), 0) != sizeof(baddr.sin_addr)) {
            close(fd);
            return -1;
        }
        break;
    case ADDR_DOMAIN:
        if (send(fd, addr, addr_len, 0) != addr_len) {
            close(fd);
            return -1;
        }
        break;
    // case ADDR_IP_V6:
    //     break;
    }

    uint16_t nsport = htons(port);
    if (send(fd, &nsport, sizeof(uint16_t), 0) != sizeof(uint16_t)) {
        close(fd);
        return -1;
    }

    /*
     *  +----+-----+-------+------+----------+----------+
     *  |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
     *  +----+-----+-------+------+----------+----------+
     *  | 1  |  1  | X'00' |  1   | Variable |    2     |
     *  +----+-----+-------+------+----------+----------+
     *
     *  if connect 0x01 even if addr type = 0x03 -> returns 10 bytes
     */
    memset(buf, 0, sizeof(buf));
    if (recv(fd, buf, 10, 0) != 10) {
        close(fd);
        return -1;
    }

    if (buf[0] != SOCKS_VERSION) {
        close(fd);
        return -1;
    }

    if (buf[1] != NO_ERROR) {
        close(fd);
        return -buf[1];
    }

    return fd;
}
