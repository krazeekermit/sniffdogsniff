#include <iostream>

#include "logging.h"
#include "rpc/sdsrpcserver.h"
#include "localnode.h"

#include "sds_config.h"
#include "net/tor.h"
#include "net/libsam3.h"

#include "webserver/sdswebuiserver.h"

#include <getopt.h>
#include <signal.h>
#include <unistd.h>

#include <vector>

using namespace std;

static struct option long_options[] = {
   {"config-file",  required_argument, 0,  'c' },
   {"log-level",    required_argument, 0,  'l' },
   {"daemon",       no_argument      , 0,  'd' },
   {0,              0,                 0,  0   }
};

struct SdsMainCtx {
    LocalNode *node;
    SdsRpcServer *rpcSrv;
    SdsWebUiServer *webSrv;

    int p2p_hidden_service;
    TorControlSession *torSession;
    Sam3Session *i2pSession;
};

static SdsMainCtx sdsContext = {
    .node = nullptr,
    .rpcSrv = nullptr,
    .webSrv = nullptr,
    .torSession = nullptr,
    .i2pSession = nullptr
};

void sigintHandler(int signo)
{
    loginfo << "INT signal received, shutting down...";
    sdsContext.rpcSrv->shutdown();
    sdsContext.node->shutdown();
    sdsContext.webSrv->shutdown();

    switch (sdsContext.p2p_hidden_service) {
    case TOR_HIDDEN_SERVICE: {

        }
        break;
    case I2P_HIDDEN_SERVICE:
        loginfo << "closing I2P session";
        sam3CloseSession(sdsContext.i2pSession);
        break;
    case NO_HIDDEN_SERVICE:
    default:
        break;
    }
}

int main(int argc, char **argv)
{

    int err;
    int optdaemon = 0;
    int optconfig = 0;
    SdsConfig cfg;
    memset(&cfg, 0, sizeof(cfg));

    int option_index;
    char c;
    while ((c = getopt_long(argc, argv, "b:c:l:d", long_options, &option_index)) > -1) {
        switch (c) {
        case 'c':
            err = sds_config_parse_file(&cfg, optarg);
            if (err > 0) {
                logfatalerr << "error parsing config file " << optarg << " at line" << err;
            } else if (err < 0) {
                logfatalerr << "error parsing config file " << optarg << " wrong data" << err;
            }
            optconfig = 1;
            //sds_config_print(&cfg);
            break;
        case 'l':
            break;
        case 'd':
            optdaemon = 1;
        default:
            break;
        }
    }

    if (!optconfig) {
        logwarn << "no configuration file submitted: using reasonable defaults";
    }

    if (optdaemon) {
        pid_t pid = fork();
        switch (pid) {
        case -1:
           perror("fork");
           exit(EXIT_FAILURE);
        case 0:
           break;
        default:
           logdebug << "sniffdogsniffd started as process pid=" << pid;
           exit(EXIT_SUCCESS);
        }
    }

    FILE *hsfp;
    char hsfpath[1024];
    char hsaddr[512];
    char *privateKey = nullptr;
    TorControlSession torSession;
    Sam3Session i2pSession;
    sdsContext.p2p_hidden_service = cfg.p2p_hidden_service;
    switch (cfg.p2p_hidden_service) {
    case TOR_HIDDEN_SERVICE: {
            tor_control_session_init(&torSession, cfg.tor_control_addr, 0, cfg.tor_auth_cookie, cfg.tor_password);


            sprintf(hsfpath, "%s/%s", cfg.work_dir_path, "onionkey.dat");
            char pkbuf[512];
            if ((hsfp = fopen(hsfpath, "r"))) {

                if (fgets(pkbuf, sizeof(pkbuf), hsfp) != NULL) {
                    privateKey = pkbuf;
                }
                fclose(hsfp);
            }

            if (tor_add_onion(&torSession, hsaddr, cfg.p2p_server_bind_addr, cfg.p2p_server_bind_port, privateKey)) {
                logfatalerr << "fatal error could not create onion hidden service" << torSession.errstr;
            }

            if ((hsfp = fopen(hsfpath, "w"))) {
                fprintf(hsfp, "%s", torSession.privkey);
                fclose(hsfp);
            }

            sdsContext.torSession = &torSession;
            loginfo << "successfully created tor hidden service at dest " << hsaddr;
        }
        break;
    case I2P_HIDDEN_SERVICE:
        sprintf(hsfpath, "%s/%s", cfg.work_dir_path, "i2pkey.dat");
        char pkbuf[512];
        if ((hsfp = fopen(hsfpath, "r"))) {

            if (fgets(pkbuf, sizeof(pkbuf), hsfp) != NULL) {
                privateKey = pkbuf;
            }
            fclose(hsfp);
        }

        if (!privateKey) {
            if (sam3GenerateKeys(&i2pSession, cfg.i2p_sam_addr, 0, Sam3SigType::EdDSA_SHA512_Ed25519))
                logfatalerr << "sam3 err" << i2pSession.error;
            privateKey = i2pSession.privkey;

            if ((hsfp = fopen(hsfpath, "w"))) {
                fprintf(hsfp, "%s", i2pSession.privkey);
                fclose(hsfp);
            }

            if (sam3CreateSession(&i2pSession, cfg.i2p_sam_addr, 0, privateKey, Sam3SessionType::SAM3_SESSION_STREAM, Sam3SigType::EdDSA_SHA512_Ed25519, nullptr)) {
                logfatalerr << "sam3 err" << i2pSession.error;
            }

            if (sam3StreamForward(&i2pSession, cfg.p2p_server_bind_addr, cfg.p2p_server_bind_port)) {
                logfatalerr << "sam3 err" << i2pSession.error;
            }

            sdsContext.i2pSession = &i2pSession;
            loginfo << "successfully created i2p hidden service at dest " << i2pSession.pubkey;
        }
        break;
    case NO_HIDDEN_SERVICE:
    default:
        break;
    }

    struct sigaction act = {0};
    act.sa_flags = SA_SIGINFO;
    act.sa_handler = &sigintHandler;
    if (sigaction(SIGINT, &act, NULL) == -1) {

    }

    //use smart ptrs?
    LocalNode  *node = new LocalNode(cfg);
    sdsContext.node = node;
    loginfo << "started tasks...";
    node->startTasks();

    SdsWebUiServer *webSrv = new SdsWebUiServer(node, "./res");
    sdsContext.webSrv = webSrv;
    loginfo << "started web ui server on ";
    webSrv->startListening("127.0.0.1", 8081, true);

    SdsRpcServer *srv = new SdsRpcServer(node);
    sdsContext.rpcSrv = srv;

    loginfo << "starting p2p server on " << cfg.p2p_server_bind_addr << ":" << cfg.p2p_server_bind_port;
    if (srv->startListening(cfg.p2p_server_bind_addr, cfg.p2p_server_bind_port)) {

    }

    delete srv;
    delete webSrv;
    delete node;
    return 0;
}
