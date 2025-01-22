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
        return errcode;
    }

    return 0;
}

static int read_reply_PROTOCOLINFO(int fd, char *method, char *cfpath) { // authmethod, cookiefile
    char reply_buf[2048];
    int tor_errno = read_reply(fd, reply_buf, sizeof(reply_buf));
    if (tor_errno)
        return tor_errno;

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
    int tor_errno = read_reply(fd, reply_buf, sizeof(reply_buf));
    if (tor_errno)
        return tor_errno;

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
    int tor_errno = read_reply(fd, reply_buf, sizeof(reply_buf));
    if (tor_errno)
        return tor_errno;

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
        return 0;
    }
    strcpy(priv, tokv);

    return 0;// serverhash, servernonce
}

char *tor_strerror(int tor_errno)
{
    if (tor_errno >= EPERM && tor_errno <= ERANGE) {
        return strerror(tor_errno);
    }

    switch (tor_errno) {
        case -2:
            return "Unknown tor control auth method";
        case 451:
            return "Resource exhausted";
        case 500:
            return "Syntax error: protocol";
        case 510:
            return "Unrecognized command";
        case 511:
            return "Unimplemented command";
        case 512:
            return "Syntax error in command argument";
        case 513:
            return "Unrecognized command argument";
        case 514:
            return "Authentication required";
        case 515:
            return "Bad authentication";
        case 550:
            return "Unspecified Tor error";
        case 551:
            return "Internal error";
        case 552:
            return "Unrecognized entity";
        case 553:
            return "Invalid configuration value";
        case 554:
            return "Invalid descriptor";
        case 555:
            return "Unmanaged entity";
        case 650:
            return "Asynchronous event notification";
        default:
            return "Unknown error";
    }
}

int tor_control_session_init(TorControlSession *ctx, char *addr, int port, int auth_cookie, char *pass)
{
    ctx->auth_cookie = auth_cookie;
    ctx->control_addr = addr;
    ctx->control_port = port;
    memset(ctx->privkey, 0, sizeof(ctx->privkey));
    memset(ctx->service_id, 0, sizeof(ctx->service_id));
    if (pass) {
        strncpy(ctx->password, pass, strlen(pass));
    }
    ctx->control_sock_fd = -1;
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
    if (fd < 0) {
        return errno;
    }

    int flags =1;
    if (setsockopt(fd, SOL_SOCKET, SO_KEEPALIVE, &flags, sizeof(flags))) {
        return errno;
    };

    if (inet_pton(AF_INET, ctx->control_addr, &address.sin_addr) <= 0) {
        return errno;
    }

    address.sin_family = AF_INET;
    address.sin_port = htons(ctx->control_port);

    if (connect(fd, (struct sockaddr*)&address, sizeof(address)) < 0) {
        return errno;
    }

    int tor_errno = 0;
    char password[256];
    size_t send_sz = 0;
    if (ctx->auth_cookie) {

        sprintf(buf, "PROTOCOLINFO\n");
        send_sz = strlen(buf);
        if (send(fd, buf, send_sz, 0) != send_sz) {
            close(fd);
            return errno;
        }
        char method[64];
        char cfpath[512];
        tor_errno = read_reply_PROTOCOLINFO(fd, method, cfpath);
        if (tor_errno) {
            close(fd);
            return tor_errno;
        }

        if (strcmp(method, "COOKIE,SAFECOOKIE")) {
            close(fd);
            return -2;
        }

        unsigned char cookie_data[32];
        FILE *cfp = fopen(cfpath, "rb");
        if (!cfp) {
            close(fd);
            return errno;
        }

        if (fread(cookie_data, sizeof(cookie_data), 1, cfp) != 1) {
            close(fd);
            fclose(cfp);
            return errno;
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
            close(fd);
            return errno;
        }

        tor_errno = read_reply_AUTHCHALLENGE(fd, server_hash, server_nonce);
        if (tor_errno) {
            close(fd);
            return tor_errno;
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
            close(fd);
            return -3;
        }

        hmac_ctx = HMAC_CTX_new();
        HMAC_Init(hmac_ctx, TOR_HMAC_CONTROLLER_TO_SERVER_KEY, strlen(TOR_HMAC_CONTROLLER_TO_SERVER_KEY), EVP_sha256());
        HMAC_Update(hmac_ctx, cookie_data, 32);
        HMAC_Update(hmac_ctx, client_nonce, 32);
        HMAC_Update(hmac_ctx, server_nonce, 32);
        HMAC_Final(hmac_ctx, server_hash_chk, &len);
        HMAC_CTX_free(hmac_ctx);

        btostrhex(password, server_hash_chk, 32);
    } else {
        strcpy(password, ctx->password);
    }

    sprintf(buf, "AUTHENTICATE %s\n", password);
    send_sz = strlen(buf);
    if (send(fd, buf, send_sz, 0) != send_sz) {
        close(fd);
        return errno;
    }

    tor_errno = read_reply(fd, buf, 2048);
    if (tor_errno) {
        close(fd);
        return tor_errno;
    }

    ctx->control_sock_fd = fd;

    if (privkey) {
        strcpy(ctx->privkey, privkey);
        sprintf(buf, "ADD_ONION %s Port=%d,%s:%d\n", privkey, bport, baddr, bport);
    } else {
        sprintf(buf, "ADD_ONION NEW:ED25519-V3 Port=%d,%s:%d\n", bport, baddr, bport);
    }
    send_sz = strlen(buf);
    if (send(fd, buf, send_sz, 0) != send_sz) {
        close(fd);
        return errno;
    }

    tor_errno = read_reply_ADD_ONION(fd, ctx->privkey, ctx->service_id);
    if (tor_errno) {
        close(fd);
        return tor_errno;
    }

    sprintf(onionaddr, "%s.onion:%d", ctx->service_id, bport);
    return 0;
}

int tor_del_onion(TorControlSession *ctx)
{
    if (ctx->control_sock_fd < 0) {
        return -1;
    }

    char buf[2048];
    sprintf(buf, "DEL_ONION %s\n", ctx->service_id);
    size_t send_sz = strlen(buf);
    if (send(ctx->control_sock_fd, buf, send_sz, 0) != send_sz) {
        return errno;
    }

    int tor_errno = read_reply(ctx->control_sock_fd, buf, 2048);
    if (tor_errno) {
        return tor_errno;
    }
    return 0;
}
