#include <iostream>

#include "logging.hpp"
#include "searchengine.h"

#include "rpc/sdsrpcserver.h"
#include "localnode.h"

#include "sds_config.h"
#include "net/tor.h"
#include "net/libsam3.h"

#include <getopt.h>

#include <vector>

using namespace std;

static struct option long_options[] = {
   {"config-file",  required_argument, 0,  'c' },
   {"log-level",    required_argument, 0,  'l' },
   {"daemon",       no_argument      , 0,  'd' },
   {0,              0,                 0,  0   }
};

int main(int argc, char **argv)
{
    int err;
    int optdaemon = 0;
    SdsConfig cfg;
    memset(&cfg, 0, sizeof(cfg));

    int option_index;
    char c;
    while ((c = getopt_long(argc, argv, "b:c:l:d", long_options, &option_index)) > -1) {
        switch (c) {
        case 'c':
            err = sds_config_parse_file(&cfg, optarg);
            if (err > 0) {
                logfatalerr(<< "error parsing config file " << optarg << " at line" << err);
            } else if (err < 0) {
            }
            sds_config_print(&cfg);
            break;
        case 'l':
            break;
        case 'b':

        default:
            break;
        }
    }

    FILE *hsfp;
    char hsfpath[1024];
    char hsaddr[512];
    char *privateKey = nullptr;
    TorControlSession torSession;
    Sam3Session i2pSession;
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
                logfatalerr(<< "fatal error could not create onion hidden service" << torSession.errstr);
            }

            if ((hsfp = fopen(hsfpath, "w"))) {
                fprintf(hsfp, "%s", torSession.privkey);
                fclose(hsfp);
            }

            loginfo(<< "successfully created tor hidden service at dest " << hsaddr);
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
                logfatalerr(<< "sam3 err" << i2pSession.error);
            privateKey = i2pSession.privkey;

            if ((hsfp = fopen(hsfpath, "w"))) {
                fprintf(hsfp, "%s", i2pSession.privkey);
                fclose(hsfp);
            }

            if (sam3CreateSession(&i2pSession, cfg.i2p_sam_addr, 0, privateKey, Sam3SessionType::SAM3_SESSION_STREAM, Sam3SigType::EdDSA_SHA512_Ed25519, nullptr)) {
                logfatalerr(<< "sam3 err" << i2pSession.error);
            }

            if (sam3StreamForward(&i2pSession, cfg.p2p_server_bind_addr, cfg.p2p_server_bind_port)) {
                logfatalerr(<< "sam3 err" << i2pSession.error);
            }

            loginfo(<< "successfully created i2p hidden service at dest " << i2pSession.pubkey);
        }
        break;
    case NO_HIDDEN_SERVICE:
    default:
        break;
    }

    //use smart ptrs?
    LocalNode  *node = new LocalNode(cfg);
    SdsRpcServer *srv = new SdsRpcServer(node);

    //start node tasks..........

    //start webui

    loginfo(<< "starting p2p server on " << cfg.p2p_server_bind_addr << ":" << cfg.p2p_server_bind_port);
    if (srv->startListening(cfg.p2p_server_bind_addr, cfg.p2p_server_bind_port)) {

    }

    return 0;
}
