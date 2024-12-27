#define TOR_CODE_SUCCESS 250
#define TOR_CODE_UNRECOGNIZED_COMMAND 510

#define TOR_HMAC_SERVER_TO_CONTROLLER_KEY "Tor safe cookie authentication server-to-controller hash"
#define TOR_HMAC_CONTROLLER_TO_SERVER_KEY "Tor safe cookie authentication controller-to-server hash"

#include "tor.h"

#include <sys/socket.h>
#include <arpa/inet.h>
#include <unistd.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>

#include <openssl/hmac.h>
#include <openssl/evp.h>

static inline void fill_rand_nonce(unsigned char *vec, size_t sz)
{
    FILE *rfp = fopen("/dev/urandom", "rb");
    if (!rfp)
        return;

    fread(vec, sizeof(uint8_t), sz, rfp);
    fclose(rfp);
}

static inline size_t btostrhex(char *out, const unsigned char *in, size_t in_sz)
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

static inline size_t strhextob(unsigned char *out, const char *in)
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

static int read_reply(int fd, char *reply_buf, size_t buf_size)
{
    size_t reply_sz = recv(fd, reply_buf, buf_size, 0);
    size_t i, v;
    char *resultp = reply_buf;
    v = 0;
    for (i = 0; i < reply_sz; i++) {
        if (reply_buf[i] == '\n') {
            reply_buf[v++] = ' ';
            if (i < reply_sz - 1)
                resultp = reply_buf + v;
        } else if (reply_buf[i] != '\r') {
            reply_buf[v++] = reply_buf[i];
        }
    }

    reply_buf[v] = '\0';

    int errcode = TOR_CODE_SUCCESS;
    sscanf(resultp, "%d ", &errcode);
    if (errcode != TOR_CODE_SUCCESS) {
        return -errcode;
    }

    return 0;
}

static int read_reply_PROTOCOLINFO(int fd, char *method, char *cfpath) { // authmethod, cookiefile
    char reply_buf[2048];
    if (read_reply(fd, reply_buf, sizeof(reply_buf)))
        return -1;

    char *tokap = NULL;
    char *tokv = NULL;
    char *toka = NULL;
    toka = strtok_r(reply_buf, " ", &tokap);
    if (!toka || strcmp(toka, "250-PROTOCOLINFO")) {
        return -1;
    }
    strtok_r(NULL, " ", &tokap);
    toka = strtok_r(NULL, " ", &tokap);
    if (!toka || strstr(toka, "250-AUTH") == NULL) {
        return -1;
    }
    toka = strtok_r(NULL, " ", &tokap);
    if (!toka) {
        return -1;
    }

    toka = strtok_r(toka, "=", &tokv);
    if (!toka || strcmp(toka, "METHODS")) {
        return -1;
    }
    strcpy(method, tokv);

    toka = strtok_r(NULL, " ", &tokap);
    if (toka) {
        toka = strtok_r(toka, "=", &tokv);
        if (toka && strcmp(toka, "COOKIEFILE") == 0) {
            *strchr(tokv + 1, '"') = '\0';
            strcpy(cfpath, tokv + 1);
        }
    }

    return 0;
}

static int read_reply_AUTHCHALLENGE(int fd, unsigned char *hash, unsigned char *nonce)
{ // authmethod, cookiefile
    char reply_buf[2048];
    if (read_reply(fd, reply_buf, sizeof(reply_buf)))
        return -1;

    char *tokap = NULL;
    char *tokv = NULL;
    char *toka = NULL;
    toka = strtok_r(reply_buf, " ", &tokap);
    toka = strtok_r(NULL, " ", &tokap);
    if (!toka || strcmp(toka, "AUTHCHALLENGE")) {
        return -1;
    }
    toka = strtok_r(NULL, " ", &tokap);
    if (!toka) {
        return -1;
    }

    toka = strtok_r(toka, "=", &tokv);
    if (!toka || strcmp(toka, "SERVERHASH")) {
        return -1;
    }
    strhextob(hash, tokv);

    toka = strtok_r(NULL, " ", &tokap);
    if (!toka) {
        return -1;
    }

    toka = strtok_r(toka, "=", &tokv);
    if (!toka || strcmp(toka, "SERVERNONCE")) {
        return -1;
    }
    strhextob(nonce, tokv);

    return 0;// serverhash, servernonce
}

static int read_reply_ADD_ONION(int fd, char *priv, char *onionaddr)
{ // authmethod, cookiefile
    char reply_buf[2048];
    if (read_reply(fd, reply_buf, sizeof(reply_buf)))
        return -1;

    char *tokap = NULL;
    char *tokv = NULL;
    char *toka = NULL;
    toka = strtok_r(reply_buf, " ", &tokap);
    toka = strtok_r(toka, "=", &tokv);
    if (!toka || strcmp(toka, "250-ServiceID")) {
        return -1;
    }
    strcpy(onionaddr, tokv);

    toka = strtok_r(NULL, " ", &tokap);
    if (!toka) {
        return -1;
    }

    toka = strtok_r(toka, "=", &tokv);
    if (!toka || strcmp(toka, "250-PrivateKey")) {
        return -1;
    }
    strcpy(priv, tokv);

    return 0;// serverhash, servernonce
}

int tor_control_session_init(TorControlSession *ctx, char *addr, int port, int auth_cookie, char *pass)
{
    ctx->auth_cookie = auth_cookie;
    ctx->control_addr = addr;
    ctx->control_port = port;
    strncpy(ctx->password, pass, strlen(pass));
    ctx->control_sock_fd = -1;
    ctx->errstr[0] = '\0';
    return 0;
}

int tor_add_onion(TorControlSession *ctx, char *onionaddr, const char *baddr, int bport, const char *privkey)
{
    int i, fd;
    int opt = 0;
    size_t valread;
    struct sockaddr_in address;
    socklen_t addrlen = sizeof(address);

    char buf[2048];

    fd = socket(AF_INET, SOCK_STREAM, 0);
    if (fd < 0)
        return -1;

    int flags =1;
    if (setsockopt(fd, SOL_SOCKET, SO_KEEPALIVE, &flags, sizeof(flags))) {
        return -1;
    };

    if (inet_pton(AF_INET, ctx->control_addr, &address.sin_addr) <= 0) {
        return -1;
    }

    address.sin_family = AF_INET;
    address.sin_port = htons(ctx->control_port);

    if (connect(fd, (struct sockaddr*)&address, sizeof(address)) < 0) {
        return -1;
    }

    char password[256];
    size_t send_sz = 0;
    if (ctx->auth_cookie) {

        sprintf(buf, "PROTOCOLINFO\n");
        send_sz = strlen(buf);
        if (send(fd, buf, send_sz, 0) != send_sz) {
            return -1;
        }
        char method[64];
        char cfpath[512];
        if (read_reply_PROTOCOLINFO(fd, method, cfpath)) {
            return -1;
        }

        if (strcmp(method, "COOKIE,SAFECOOKIE")) {
            return -1;
        }

        unsigned char cookie_data[32];
        FILE *cfp = fopen(cfpath, "rb");
        if (!cfp) {
            sprintf(ctx->errstr, "cannot open tor cookie file %s: %s", cfpath, strerror(errno));
            return -1;
        }

        if (fread(cookie_data, sizeof(cookie_data), 1, cfp) != 1) {
            fclose(cfp);
            return -1;
        }
        fclose(cfp);

        unsigned char client_nonce[32];
        unsigned char  server_hash[32];
        unsigned char  server_hash_chk[32];
        unsigned char  server_nonce[32];
        fill_rand_nonce(client_nonce, 32);

        char  hex_data[65];
        btostrhex(hex_data, client_nonce, 32);
        sprintf(buf, "AUTHCHALLENGE SAFECOOKIE %s\n", hex_data);
        send_sz = strlen(buf);
        if (send(fd, buf, send_sz, 0) != send_sz) {
            return -1;
        }

        if (read_reply_AUTHCHALLENGE(fd, server_hash, server_nonce)) {
            return -1;
        }

        HMAC_CTX *hmac_ctx = HMAC_CTX_new();
        unsigned int  len;
        HMAC_Init(hmac_ctx, TOR_HMAC_SERVER_TO_CONTROLLER_KEY, strlen(TOR_HMAC_SERVER_TO_CONTROLLER_KEY), EVP_sha256());
        HMAC_Update(hmac_ctx, cookie_data, 32);
        HMAC_Update(hmac_ctx, client_nonce, 32);
        HMAC_Update(hmac_ctx, server_nonce, 32);
        HMAC_Final(hmac_ctx, server_hash_chk, &len);
        HMAC_CTX_free(hmac_ctx);

        if (memcmp(server_hash, server_hash_chk, 32)) {
            return -1;
        }

        hmac_ctx = HMAC_CTX_new();
        HMAC_Init(hmac_ctx, TOR_HMAC_CONTROLLER_TO_SERVER_KEY, strlen(TOR_HMAC_CONTROLLER_TO_SERVER_KEY), EVP_sha256());
        HMAC_Update(hmac_ctx, cookie_data, 32);
        HMAC_Update(hmac_ctx, client_nonce, 32);
        HMAC_Update(hmac_ctx, server_nonce, 32);
        HMAC_Final(hmac_ctx, server_hash_chk, &len);
        HMAC_CTX_free(hmac_ctx);

        btostrhex(password, server_hash_chk, 32);
    } else if (ctx->password) {
        strcpy(password, ctx->password);
    }

    sprintf(buf, "AUTHENTICATE %s\n", password);
    send_sz = strlen(buf);
    if (send(fd, buf, send_sz, 0) != send_sz) {
        return -1;
    }
    if (read_reply(fd, buf, 2048)) {
        return -1;
    }

    ctx->control_sock_fd = fd;

    if (privkey) {
        sprintf(buf, "ADD_ONION %s Port=%d,%s:%d\n", privkey, bport, baddr, bport);
    } else {
        sprintf(buf, "ADD_ONION NEW:ED25519-V3 Port=%d,%s:%d\n", bport, baddr, bport);
    }
    send_sz = strlen(buf);
    if (send(fd, buf, send_sz, 0) != send_sz) {
        return -1;
    }

    if (read_reply_ADD_ONION(fd, ctx->privkey, onionaddr)) {
        return -1;
    }

    return 0;
}

int tor_del_onion(TorControlSession *ctx, const char *onion_addr)
{
    if (ctx->control_sock_fd < 0) {
        return -1;
    }

    char buf[2048];
    sprintf(buf, "DEL_ONION %s\n", onion_addr);
    size_t send_sz = strlen(buf);
    if (send(ctx->control_sock_fd, buf, send_sz, 0) != send_sz) {
        return -1;
    }
    if (read_reply(ctx->control_sock_fd, buf, 2048)) {
        return -1;
    }
    return 0;
}
