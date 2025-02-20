#include "netutil.h"

#include <stdint.h>
#include <string.h>
#include <endian.h>
#include <stdlib.h>

#include <arpa/inet.h>
#include <sys/socket.h>
#include <fcntl.h>
#include <errno.h>

#include <openssl/md5.h>
#include <openssl/hmac.h>

#define STUN_HEADER_LEN                  20
#define STUN_TRANSACTION_ID_SIZE         12

#define STUN_MSGCLASS_REQUEST            0x0000
#define STUN_MSGCLASS_INDICATION         0x0110
#define STUN_MSGCLASS_SUCCESS_RESPONSE   0x0100
#define STUN_MSGCLASS_ERROR_RESPONSE     0x0110

#define STUN_METHOD_BIND_REQUEST         0x0001

#define STUN_MAGIC_COOKIE                0x2112A442

#define STUN_ATTRIBUTE_MAPPED_ADDR       0x0001
#define STUN_ATTRIBUTE_XOR_MAPPED_ADDR   0x0020
#define STUN_ATTRIBUTE_USERNAME          0x0006
#define STUN_ATTRIBUTE_REALM             0x0014
#define STUN_ATTRIBUTE_MESSAGE_INTEGRITY 0x0008

struct stun_header {
    uint16_t msg_class;
    uint16_t method;
    uint16_t msg_len;
    uint32_t magic_cookie;
    uint8_t txid[STUN_TRANSACTION_ID_SIZE];
};

struct stun_attribute {
    uint16_t attr_type;
    uint16_t value_len;
    uint8_t *value;
};

struct stun_message {
    struct stun_header header;
    int attrs_capacity;
    int attrs_len;
    struct stun_attribute *attrs;
};

static inline struct stun_message stun_message_create()
{
    struct stun_message msg;
    memset(&msg, 0, sizeof(msg));
    return msg;
}

static void stun_message_free(struct stun_message *msg)
{
    int i;
    for (i = 0; i < msg->attrs_len; i++) {
        free(msg->attrs[i].value);
    }
    free(msg->attrs);
}

struct stun_attribute *stun_message_add_attribute(struct stun_message *msg, struct stun_attribute attr)
{
    if (msg->attrs && msg->attrs_capacity >= msg->attrs_len) {
        msg->attrs_capacity *= 2;
        msg->attrs = (struct stun_attribute*) realloc(msg->attrs, msg->attrs_capacity * sizeof(struct stun_attribute));
    } else {
        msg->attrs_capacity = 2;
        msg->attrs = (struct stun_attribute*) malloc(msg->attrs_capacity * sizeof(struct stun_attribute));
    }

    msg->attrs[msg->attrs_len] = attr;
    return &msg->attrs[msg->attrs_len++];
}

struct stun_attribute *stun_message_get_attribute(struct stun_message *msg, uint16_t attr_type)
{
    if (msg->attrs) {
        int i;
        for (i = 0; i < msg->attrs_len; i++) {
            if (msg->attrs[i].attr_type == attr_type) {
                return &(msg->attrs[i]);
            }
        }
    }

    return NULL;
}

static void stun_message_write_header(uint8_t *buffer, struct stun_header *header)
{
    uint16_t method = htons(header->method);
    memcpy(buffer, &method, sizeof(method));
    buffer += sizeof(method);

    uint16_t msg_len = htons(header->msg_len);
    memcpy(buffer, &msg_len, sizeof(msg_len));
    buffer += sizeof(msg_len);

    uint32_t mcbe = htonl(STUN_MAGIC_COOKIE);
    memcpy(buffer, &mcbe, sizeof(mcbe));
    buffer += sizeof(mcbe);

    buffer[0] = header->txid[0];
    buffer[1] = header->txid[1];
    buffer[2] = header->txid[2];
    buffer[3] = header->txid[3];
    buffer[4] = header->txid[4];
    buffer[5] = header->txid[5];
    buffer[6] = header->txid[6];
    buffer[7] = header->txid[7];
    buffer[8] = header->txid[8];
    buffer[9] = header->txid[9];
    buffer[10] = header->txid[10];
    buffer[11] = header->txid[11];
}

static void stun_write_message(uint8_t **bufp, size_t *buf_len, struct stun_message *msg)
{
    int i;

    *buf_len = STUN_HEADER_LEN;
    for (i = 0; i < msg->attrs_len; i++) {
        *buf_len += 2*sizeof(uint16_t);
        *buf_len += msg->attrs[i].value_len;
        *buf_len += msg->attrs[i].value_len % 4; //padding
    }

    uint8_t *buffer = (uint8_t*) malloc(*buf_len * sizeof(uint8_t));
    *bufp = buffer;

    buffer += STUN_HEADER_LEN;
    for (i = 0; i < msg->attrs_len; i++) {
        struct stun_attribute attr = msg->attrs[i];

        memcpy(buffer, &attr.attr_type, sizeof(attr.attr_type));
        buffer += sizeof(attr.attr_type);

        uint16_t beval_len = htons(attr.value_len);
        memcpy(buffer, &beval_len, sizeof(beval_len));
        buffer += sizeof(beval_len);

        memcpy(buffer, attr.value, attr.value_len);
        buffer += attr.value_len;

        /*
           Since STUN aligns attributes on 32-bit boundaries, attributes whose content
           is not a multiple of 4 bytes are padded with 1, 2, or 3 bytes of
           padding so that its value contains a multiple of 4 bytes.  The
           padding bits are ignored, and may be any value.
        */
        int padding = attr.value_len % 4;
        if (padding > 0) {
            memset(buffer, 0, padding);
            buffer += padding;
        }
    }

    msg->header.msg_len = *buf_len - STUN_HEADER_LEN;
    stun_message_write_header(*bufp, &msg->header);
}

int stun_read_message_header(struct stun_header *header, u_int8_t *buffer, size_t buf_len)
{
    uint16_t ht = 0;
    memcpy(&ht, buffer, sizeof(ht));
    buffer += sizeof(ht);

    header->msg_class = ht & 0x0110;
    header->method = ht & 0xfeef;

    uint16_t msg_len = 0;
    memcpy(&msg_len, buffer, sizeof(msg_len));
    buffer += sizeof(msg_len);

    header->msg_len = ntohs(msg_len);

    memcpy(&header->magic_cookie, buffer, sizeof(header->magic_cookie));
    buffer += sizeof(header->magic_cookie);

    header->txid[0] = buffer[0];
    header->txid[1] = buffer[1];
    header->txid[2] = buffer[2];
    header->txid[3] = buffer[3];
    header->txid[4] = buffer[4];
    header->txid[5] = buffer[5];
    header->txid[6] = buffer[6];
    header->txid[7] = buffer[7];
    header->txid[8] = buffer[8];
    header->txid[9] = buffer[9];
    header->txid[10] = buffer[10];
    header->txid[11] = buffer[11];
}

int stun_read_message_arributes(struct stun_message *msg, u_int8_t *buffer, size_t buf_len)
{
    uint8_t *bufend = buffer + buf_len;
    int i;
    while (buffer < bufend) {
        struct stun_attribute attr;

        uint16_t type = 0;
        memcpy(&type, buffer, sizeof(attr.attr_type));
        buffer += sizeof(attr.attr_type);
        attr.attr_type = ntohs(type);

        uint16_t beval_len = 0;
        memcpy(&beval_len, buffer, sizeof(beval_len));
        buffer += sizeof(beval_len);

        attr.value_len = ntohs(beval_len);

        attr.value = (uint8_t*) malloc(attr.value_len * sizeof(uint8_t));
        memcpy(attr.value, buffer, attr.value_len);
        buffer += attr.value_len + attr.value_len % 4;

        stun_message_add_attribute(msg, attr);
    }

    return 0;
}

static void stun_message_add_mapped_address_attribute(struct stun_message *msg, struct sockaddr_in *addr)
{
    struct stun_attribute attr;
    attr.attr_type = STUN_ATTRIBUTE_MAPPED_ADDR;

    attr.value_len = 4 + sizeof(addr->sin_addr.s_addr);
    attr.value = (uint8_t*) malloc(attr.value_len * sizeof(uint8_t));
    attr.value[0] = 0x00;
    attr.value[1] = (addr->sin_family == AF_INET6 ? 0x02 : 0x01);

    memcpy(attr.value + 2, &addr->sin_port, sizeof(in_port_t));
    memcpy(attr.value + sizeof(in_port_t), &addr->sin_addr.s_addr, sizeof(in_addr_t));

    stun_message_add_attribute(msg, attr);
}

static int stun_message_get_mapped_address_attribute(struct stun_message *msg, struct sockaddr_in *addr)
{
    struct stun_attribute *attr = stun_message_get_attribute(msg, STUN_ATTRIBUTE_MAPPED_ADDR);
    if (attr) {
        addr->sin_family = attr->value[1] == 0x02 ? AF_INET6 : AF_INET;
        memcpy(&addr->sin_port, attr->value + 2, sizeof(in_port_t));
        memcpy(&addr->sin_addr.s_addr, attr->value + sizeof(in_port_t) + 2, sizeof(in_addr_t));
        return 0;
    }

    return -1;
}

static void stun_message_add_xor_mapped_address_attribute(struct stun_message *msg, struct sockaddr_in *addr)
{
    struct stun_attribute attr;
    attr.attr_type = STUN_ATTRIBUTE_XOR_MAPPED_ADDR;

    attr.value_len = 4 + sizeof(addr->sin_addr.s_addr);
    attr.value = (uint8_t*) malloc(attr.value_len * sizeof(uint8_t));
    attr.value[0] = 0x00;
    attr.value[1] = (addr->sin_family == AF_INET6 ? 0x02 : 0x01);

    uint16_t xport = addr->sin_port ^ htons(msg->header.magic_cookie >> 16);
    memcpy(attr.value + 2, &xport, sizeof(in_port_t));

    in_addr_t xaddr = addr->sin_addr.s_addr ^ htonl(msg->header.magic_cookie);
    memcpy(attr.value + sizeof(in_port_t), &xaddr, sizeof(in_addr_t));

    stun_message_add_attribute(msg, attr);
}

static int stun_message_get_xor_mapped_address_attribute(struct stun_message *msg, struct sockaddr_in *addr)
{
    struct stun_attribute *attr = stun_message_get_attribute(msg, STUN_ATTRIBUTE_XOR_MAPPED_ADDR);
    if (attr) {
        addr->sin_family = attr->value[1] == 0x02 ? AF_INET6 : AF_INET;

        uint16_t xport = 0;
        memcpy(&xport, attr->value + 2, sizeof(in_port_t));
        addr->sin_port = addr->sin_port ^ htons(msg->header.magic_cookie >> 16);

        in_addr_t xaddr = 0;
        memcpy(&xaddr, attr->value + sizeof(in_port_t) + 2, sizeof(in_addr_t));
        addr->sin_addr.s_addr = xaddr ^ htonl(msg->header.magic_cookie);
    }

    return -1;
}

static void stun_message_add_username_attribute(struct stun_message *msg, const char *username, const char *password)
{
    struct stun_attribute attr;
    attr.attr_type = STUN_ATTRIBUTE_USERNAME;

    struct stun_attribute *realm = stun_message_get_attribute(msg, STUN_ATTRIBUTE_REALM);
    if (realm) {
        uint8_t username[513];
        snprintf((char*) username, 513, "%s:%s:%s", username, realm->value, password);
        stun_message_add_attribute(msg, attr);
    }
}

static int stun_message_get_username_attribute(struct stun_message *msg)
{
    struct stun_attribute *attr = stun_message_get_attribute(msg, STUN_ATTRIBUTE_USERNAME);
    if (attr) {

    }

    return -1;
}

static void stun_message_add_message_integrity_attribute(struct stun_message *msg)
{
    struct stun_attribute attr;
    attr.attr_type = STUN_ATTRIBUTE_MESSAGE_INTEGRITY;

    struct stun_attribute *username = stun_message_get_attribute(msg, STUN_ATTRIBUTE_USERNAME);
    if (username) {
        uint8_t key[MD5_DIGEST_LENGTH];
        MD5(username->value, username->value_len, key);

        struct stun_attribute *mi = stun_message_add_attribute(msg, attr);
        mi->value_len = MD5_DIGEST_LENGTH;
        mi->value = (uint8_t*) malloc(mi->value_len * sizeof(uint8_t));
        memset(mi->value, 0, MD5_DIGEST_LENGTH);

        uint8_t *buf = NULL;
        size_t buf_len = 0;
        stun_write_message(&buf, &buf_len, msg);

        HMAC_CTX *hmac_ctx = HMAC_CTX_new();
        unsigned int  len;
        HMAC_Init(hmac_ctx, key, MD5_DIGEST_LENGTH, EVP_md5());
        HMAC_Update(hmac_ctx, buf, buf_len);
        HMAC_Final(hmac_ctx, mi->value, &len);
        HMAC_CTX_free(hmac_ctx);

        free(buf);
    }
}

/*
    STUN Bind request (TCP)
*/
int stun_bind_request(const char *stun_addr, int stun_port, struct sockaddr_in *reflexive_addr)
{
    struct stun_message request_msg = stun_message_create();
    request_msg.header.magic_cookie = STUN_MAGIC_COOKIE;
    request_msg.header.msg_class = STUN_MSGCLASS_REQUEST;
    request_msg.header.method = STUN_METHOD_BIND_REQUEST;
    memset(request_msg.header.txid+0, 'a', 3);
    memset(request_msg.header.txid+3, '4', 3);
    memset(request_msg.header.txid+6, 'e', 3);
    memset(request_msg.header.txid+9, 'r', 3);
    request_msg.header.msg_len = 0;

    int fd = net_socket_connect(stun_addr, stun_port, 0);
    if (fd < 1) {
        return -1;
    }

    uint8_t *buf = NULL;
    size_t buf_len = 0;
    stun_write_message(&buf, &buf_len, &request_msg);

    send(fd, buf, buf_len, 0);
    free(buf);
    stun_message_free(&request_msg);

    struct stun_message response_msg = stun_message_create();

    buf = (uint8_t*) malloc(STUN_HEADER_LEN * sizeof(uint8_t));
    size_t nrecv = recv(fd, buf, STUN_HEADER_LEN * sizeof(uint8_t), 0);

    stun_read_message_header(&response_msg.header, buf, nrecv);

    int ret = 0;
    if (response_msg.header.msg_class != STUN_MSGCLASS_SUCCESS_RESPONSE) {
        ret = -1;
        goto end_bind_req;
    }

    if (response_msg.header.msg_len > 0) {
        buf = (uint8_t*) realloc(buf, response_msg.header.msg_len * sizeof(uint8_t));

        nrecv = recv(fd, buf, response_msg.header.msg_len * sizeof(uint8_t), 0);
        stun_read_message_arributes(&response_msg, buf, nrecv);
        if (stun_message_get_mapped_address_attribute(&response_msg, reflexive_addr)) {
            if (stun_message_get_xor_mapped_address_attribute(&response_msg, reflexive_addr)) {
                ret = -1;
                goto end_bind_req;
            }
        }

    }

end_bind_req:
    stun_message_free(&response_msg);
    return ret;
}

//int main(int argc, char** argv)
//{
//    struct sockaddr_in reflexive_addr;
//    stun_bind_request(argv[1], atoi(argv[2]), &reflexive_addr);
//    fprintf(stderr, "reflex_addr = %s:%d\n", inet_ntoa(reflexive_addr.sin_addr), ntohs(reflexive_addr.sin_port));
//    while(1) {}
//}
