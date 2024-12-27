#ifndef TOR_H
#define TOR_H

#ifdef __cplusplus
extern "C" {
#endif

typedef struct {
    char *control_addr;
    int control_port;
    int auth_cookie;
    char password[512];
    int control_sock_fd;
    char privkey[512];
    char errstr[512];
} TorControlSession;

int tor_control_session_init(TorControlSession *ctx, char *addr, int port, int auth_cookie, char *pass);

int tor_add_onion(TorControlSession *ctx, char *onionaddr, const char *baddr, int bport, const char *privkey);
int tor_del_onion(TorControlSession *ctx, const char *onionaddr);

#ifdef __cplusplus
}
#endif

#endif // TOR_H
