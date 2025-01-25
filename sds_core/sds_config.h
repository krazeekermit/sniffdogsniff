#ifndef SDSCONFIG_H
#define SDSCONFIG_H

#include "crawler/searchengine.h"

#include <vector>

#define NO_HIDDEN_SERVICE  0
#define TOR_HIDDEN_SERVICE 1
#define I2P_HIDDEN_SERVICE 2

struct SdsConfig {
    char *work_dir_path;
    char *log_file_name;
    int log_to_file;
    int db_cache_sz;
    int allow_result_invalidate;
    char *web_ui_bind_addr;
    std::vector<char*> known_peers;
    std::vector<char*> blacklisted_peers;
    int force_tor_proxy;
    char *tor_socks5_addr;
    int tor_socks5_port;
    char *p2p_server_bind_addr;
    int p2p_server_bind_port;
    int p2p_hidden_service;

    char *tor_control_addr;
    int tor_control_port;
    int tor_auth_cookie;
    char *tor_password;
    char *tor_cookie_file_path;

    char *i2p_sam_addr;
    char i2p_sam_port;
    char *i2p_sam_user;
    char *i2p_sam_password;

    std::vector<SearchEngineConfigs> search_engines;
};

int sds_config_parse_file(SdsConfig *cfg, const char *path);

void sds_config_print(SdsConfig *cfg);

#endif // SDSCONFIG_H
