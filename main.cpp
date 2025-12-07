#include <iostream>

#include "common/loguru.hpp"
#include "p2p/sdsp2pserver.h"
#include "sds_core/localnode.h"
#include "sds_core/sdsconfigfile.h"

#include "net/tor.h"
#include "net/libsam3.h"

#include "webserver/sdswebuiserver.h"

#include <getopt.h>
#include <signal.h>
#include <unistd.h>
#include <sys/stat.h>

#include <vector>

using namespace std;

static struct option long_options[] = {
   {"config-file",  required_argument, 0,  'c' },
   {"log-level",    required_argument, 0,  'l' },
   {"daemon",       no_argument      , 0,  'd' },
   {0,              0,                 0,  0   }
};

SdsP2PServer *p2pSrv = nullptr;

void sigintHandler(int signo)
{
    LOG_F(INFO, "INT signal received, shutting down...");
    p2pSrv->shutdown();
}

int main(int argc, char **argv)
{
    int err;
    int optdaemon = 0;
    int optconfig = 0;
    SdsConfigFile *cfgFile = new SdsConfigFile();
    loguru::init(argc, argv);

    int option_index;
    char c;
    while ((c = getopt_long(argc, argv, "b:c:v:d", long_options, &option_index)) > -1) {
        switch (c) {
        case 'c':
            err = cfgFile->parse(optarg);
            if (err > 0) {
                LOG_F(FATAL, "error parsing config file %s at line %d", optarg, err);
            } else if (err < 0) {
                LOG_F(FATAL, "error parsing config file %s at line %d", optarg, err);
            }

            LOG_S(1) << "configuration file: " << cfgFile;
            optconfig = 1;
            break;
        case 'd':
            optdaemon = 1;
        default:
            break;
        }
    }

    if (!optconfig) {
        LOG_F(WARNING, "no configuration file submitted: using reasonable defaults");
    }

    FILE *hsfp;
    char hsfpath[1024];
    bool logToFile = cfgFile->getDefaultSection()->lookupBool("log_to_file");
    std::string workDirPath = cfgFile->getDefaultSection()->lookupString("work_dir_path");
    if (optdaemon) {
        std::string pidFilePath = workDirPath + "/sds.pid";
        struct stat pidstat = {0};
        if (!stat(pidFilePath.c_str(), &pidstat)) {
            LOG_F(FATAL, "Another instance of %s is already running, pidfile=%s", argv[0], hsfpath);
        }
        pid_t pid = fork();
        switch (pid) {
        case -1:
           LOG_F(FATAL, "error creating child process");
        case 0:
            logToFile = true;
           break;
        default:
           hsfp = fopen(hsfpath, "w");
           if (!hsfp) {
               LOG_F(FATAL, "%s: unable to write pid file", hsfpath);
           }
           fprintf(hsfp, "%d", pid);
           fclose(hsfp);
           LOG_F(INFO, "started as daemon pid=%d", pid);
           exit(EXIT_SUCCESS);
        }
    }

    if (logToFile) {
        std::string logFilePath = workDirPath + "/" + cfgFile->getDefaultSection()->lookupString("log_file_name", "sds.log");
        loguru::add_file(logFilePath.c_str(), loguru::Truncate, loguru::Verbosity_MAX);
    }

    LocalNode  *node = new LocalNode(cfgFile);

    bool torHiddenService = cfgFile->getDefaultSection()->lookupBool("tor_hidden_service");
    bool i2pHiddenService = cfgFile->getDefaultSection()->lookupBool("i2p_hidden_service");

    std::string p2pServerBindAddr = cfgFile->getDefaultSection()->lookupString("p2p_bind_addr", "127.0.0.1");
    int p2pServerBindPort = cfgFile->getDefaultSection()->lookupInt("p2p_bind_port", 4111);

    if (torHiddenService && i2pHiddenService) {
        LOG_F(FATAL, "can't use both i2p and tor hidden service!");
    }

    char hsaddr[512];
    char *privateKey = nullptr;
    TorControlSession torSession;
    Sam3Session i2pSession;
    if (torHiddenService) {
        tor_control_session_init(
            &torSession,
            cfgFile->getDefaultSection()->lookupString("tor_control_addr").c_str(),
            cfgFile->getDefaultSection()->lookupInt("tor_control_port"),
            cfgFile->getDefaultSection()->lookupBool("tor_control_auth_cookie") ? 1 : 0,
            cfgFile->getDefaultSection()->lookupString("tor_control_password").c_str()
        );

        std::string torKeyPath = workDirPath + "/onionkey.dat";
        char pkbuf[512];
        if ((hsfp = fopen(torKeyPath.c_str(), "r"))) {
            if (fgets(pkbuf, sizeof(pkbuf), hsfp) != NULL) {
                privateKey = pkbuf;
            }
            fclose(hsfp);
        }

        int tor_errno = tor_add_onion(&torSession, hsaddr, p2pServerBindAddr.c_str(), p2pServerBindPort, privateKey);
        if (tor_errno) {
            LOG_F(FATAL, "could not create tor hidden service: %s", tor_strerror(tor_errno));
        }

        if ((hsfp = fopen(torKeyPath.c_str(), "w"))) {
            fprintf(hsfp, "%s", torSession.privkey);
            fclose(hsfp);
        }

        node->setSelfNodeAddress(hsaddr);
        LOG_F(INFO, "successfully created tor hidden service at dest %s", hsaddr);
    } else if (i2pHiddenService) {
        std::string i2pSamAddr = cfgFile->getDefaultSection()->lookupString("i2p_sam_addr");
        int i2pSamPort = cfgFile->getDefaultSection()->lookupInt("i2p_sam_port");

        std::string i2pKeyPath = workDirPath + "/i2pkey.dat";
        char pkbuf[SAM3_PRIVKEY_MAX_SIZE];
        memset(pkbuf, 0, sizeof(pkbuf));
        if ((hsfp = fopen(i2pKeyPath.c_str(), "r"))) {
            if (fgets(pkbuf, sizeof(pkbuf), hsfp) != NULL) {
                privateKey = pkbuf;
            }
            fclose(hsfp);
        }

        if (!privateKey) {
            if (sam3GenerateKeys(&i2pSession, i2pSamAddr.c_str(), i2pSamPort, Sam3SigType::EdDSA_SHA512_Ed25519)) {
                LOG_F(FATAL, "could not create i2p hidden service: %s", i2pSession.error);
            }

            strcpy(pkbuf, i2pSession.privkey);
            privateKey = pkbuf;
            if ((hsfp = fopen(i2pKeyPath.c_str(), "w"))) {
                fprintf(hsfp, "%s", i2pSession.privkey);
                fclose(hsfp);
            }
        }

        if (sam3CreateSession(&i2pSession, i2pSamAddr.c_str(), i2pSamPort, privateKey, Sam3SessionType::SAM3_SESSION_STREAM, Sam3SigType::EdDSA_SHA512_Ed25519, nullptr)) {
            LOG_F(FATAL, "could not create i2p hidden service: ", i2pSession.error);
        }

        if (sam3StreamForward(&i2pSession, p2pServerBindAddr.c_str(), p2pServerBindPort)) {
            LOG_F(FATAL, "could not create i2p hidden service: ", i2pSession.error);
        }

        /*
         *  As of now we use the public key as an address (base64) thus the naming lookup is not necessary.
         *  In the future we can create base32 i2p encoded addresses
         */
        sprintf(hsaddr, "%s.i2p:%d", i2pSession.pubkey, p2pServerBindPort);
        node->setSelfNodeAddress(hsaddr);
        LOG_F(INFO, "successfully created i2p hidden service at dest %s", hsaddr);
    } else {
        sprintf(hsaddr, "%s:%d", p2pServerBindAddr.c_str(), p2pServerBindPort);
        node->setSelfNodeAddress(hsaddr);
    }

    struct sigaction act = {0};
    act.sa_flags = SA_SIGINFO;
    act.sa_handler = &sigintHandler;
    if (sigaction(SIGINT, &act, NULL) == -1) {

    }

    LOG_F(INFO, "started tasks...");
    node->startTasks();

    SdsWebUiServer *webSrv = new SdsWebUiServer(node, "./res");

    std::string webUIBindAddr = cfgFile->getDefaultSection()->lookupString("web_ui_bind_addr", "127.0.0.1");
    int webUIBindPort = cfgFile->getDefaultSection()->lookupInt("web_ui_bind_port", 8081);
    LOG_F(INFO, "started web ui server on %s:%d", webUIBindAddr.c_str(), webUIBindPort);
    webSrv->startListening(webUIBindAddr.c_str(), webUIBindPort, true);

    p2pSrv = new SdsP2PServer(node);

    LOG_F(INFO, "starting p2p server on %s:%d", p2pServerBindAddr.c_str(), p2pServerBindPort);
    if (p2pSrv->startListening(p2pServerBindAddr.c_str(), p2pServerBindPort)) {
        LOG_F(ERROR, "cant start p2p server");
    }

    node->shutdown();
    webSrv->shutdown();

    if (torHiddenService) {
        LOG_F(INFO, "deleting tor hidden service at dest %s", hsaddr);
        int tor_errno = tor_del_onion(&torSession);
        if (tor_errno) {
            LOG_F(FATAL, "could not delete tor hidden service: ", tor_strerror(tor_errno));
        }
    } else if (i2pHiddenService) {
        LOG_F(INFO, "deleting i2p hidden service at dest %s", hsaddr);
        sam3CloseSession(&i2pSession);
    }

    delete p2pSrv;
    delete webSrv;
    delete node;
    delete cfgFile;

    LOG_F(INFO, "shutdown complete, goodbye!");

    return 0;
}
