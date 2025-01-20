#include "sds_config.h"

#include <iostream>
#include <cstring>
#include <vector>

#define SERVICE_RPC_PORT                    "service_rpc_port"
#define WORK_DIR_PATH                       "work_dir_path"
#define LOG_FILE_NAME                       "log_file_name"
#define LOG_TO_FILE                         "log_to_file"
#define ALLOW_RESULTS_INVALIDATION          "allow_results_invalidation"
#define DB_CACHE_SIZE                       "db_cache_size"
#define PEERS                               "peers"
#define BLACKLIST_PEERS                     "blacklist_peers"
#define ID                                  "id"
#define ADDRESS                             "address"
#define EXTERNAL_SEARCH_ENGINE              "external_search_engine"
#define NAME                                "name"
#define USER_AGENT                          "user_agent"
#define SEARCH_QUERY_URL                    "search_query_url"
#define RESULTS_CONTAINER_ELEMENT           "results_container_element"
#define RESULT_CONTAINER_ELEMENT            "result_container_element"
#define RESULT_URL_ELEMENT                  "result_url_element"
#define RESULT_URL_PROPERTY                 "result_url_property"
#define RESULT_URL_IS_JSON                  "result_url_is_json"
#define RESULT_URL_JSON_PROPERTY            "result_url_json_property"
#define RESULT_TITLE_ELEMENT                "result_title_element"
#define RESULT_TITLE_PROPERTY               "result_title_property"
#define PROVIDED_DATA_TYPE                  "provided_data_type"
#define WEB_UI_BIND_ADDR                    "web_ui_bind_addr"
#define PROXY_SETTINGS                      "proxy_settings"
#define FORCE_TOR_PROXY                     "force_tor_proxy"
#define TOR_SOCKS5_ADDR                     "tor_socks5_addr"
#define TOR_SOCKS5_PORT                     "tor_socks5_port"
#define P2P_HIDDEN_SERVICE                  "p2p_hidden_service"
#define NONE                                "none"
#define TOR                                 "tor"
#define TOR_CONTROL_ADDR                    "tor_control_addr"
#define TOR_CONTROL_PORT                    "tor_control_port"
#define TOR_CONTROL_PASSWORD                "tor_control_password"
#define TOR_CONTROL_AUTH_COOKIE             "tor_control_auth_cookie"
#define TOR_CONTROL_COOKIE_FILE             "tor_cookie_file"
#define I2P                                 "i2p"
#define I2P_SAM_ADDR                        "i2p_sam_addr"
#define I2P_SAM_PORT                        "i2p_sam_port"
#define I2P_SAM_USER                        "i2p_sam_user"
#define I2P_SAM_PASSWORD                    "i2p_sam_password"
#define P2P_BIND_ADDR                       "p2p_bind_addr"
#define P2P_BIND_PORT                       "p2p_bind_port"

#define DEFAULT_LOG_FILE_NAME               "sds.log"

static char *strlrtrim(char *s)
{
    int l = strlen(s);
    char *sep = s + l - 1;
    for (; s < sep && (*s == ' ' || *s == '\t'); s++);
    while ((*sep == ' ' || *sep == '\n' || *sep == '\t'))
        *sep = '\0';

    return s;
}

static char *strclone(const char *s) {
    if (!s)
        return nullptr;

    long len = strlen(s) + 1;
    char *clone = new char[len];
    memcpy(clone, s, len);
    return clone;
}

struct cfgentry {
    char *key;
        char *value;
        std::vector<cfgentry*> *list;
};

static cfgentry *cfgentry_new()
{
    cfgentry *e = new cfgentry();
    e->key = nullptr;
    e->list = nullptr;
    e->value = nullptr;
    return e;
}

static void cfgentry_free(cfgentry *e)
{
    if (!e)
        return;

    if (e->key)
        delete[] e->key;

    if (e->value)
        delete[] e->value;

    if (e->list) {
        int i;
        int sz = e->list->size();
        for (i = 0; i < sz; i++) {
            cfgentry *s = e->list->at(i);
            cfgentry_free(s);
        }

        delete e->list;
    }

    delete e;
}

static cfgentry *lookup(cfgentry *e, const char *key)
{
    int i;
    for (i = 0; i < e->list->size(); i++) {
        cfgentry *ie = e->list->at(i);
        if (!strcmp(ie->key, key))
            return ie;
    }
    return nullptr;
}

static int lookups(cfgentry *e, std::vector<cfgentry*> &list, const char *key)
{
    list.clear();
    int i;
    for (i = 0; i < e->list->size(); i++) {
        cfgentry *ie = e->list->at(i);
        if (!strcmp(ie->key, key))
            list.push_back(ie);
    }
    return list.size();
}

int lookup_bool(cfgentry *e, const char *key, int dvalue)
{
    cfgentry *ie = lookup(e, key);
    if (ie) {
        return strcmp(ie->value, "true") == 0 || strcmp(ie->value, "yes") == 0;
    }
    return dvalue;
}

int lookup_int(cfgentry *e, const char *key, int dvalue)
{
    cfgentry *ie = lookup(e, key);
    if (ie) {
        return atoi(ie->value);
    }
    return dvalue;
}

char *lookup_string(cfgentry *e, const char *key, const char *dvalue)
{
    cfgentry *ie = lookup(e, key);
    if (ie) {
        return strclone(ie->value);
    }
    return strclone(dvalue);
}

int lookup_strings(cfgentry *e, std::vector<char*> &list, const char *key)
{
    int i;
    std::vector<cfgentry*> entries;

    list.clear();
    lookups(e, entries, key);
    for (i = 0; i < entries.size(); i++) {
        list.push_back(strclone(entries[i]->value));
    }

    return list.size();
}

static int parse(FILE *fp, cfgentry *r)
{
    int lineno = 0;
    char buf[512];
    char *keyp = nullptr, *valuep = nullptr;
    cfgentry *e = r;
    while (fgets(buf, sizeof(buf), fp)) {
        lineno++;
        keyp = strchr(buf, '#');
        if (keyp)
            *keyp = '\0';
        strlrtrim(buf);
        if (buf[0] == '\0')
            continue;

        cfgentry *ie = cfgentry_new();
        keyp = strchr(buf, '[');
        valuep = strchr(buf, ']');
        if (keyp && valuep) {
            *valuep = '\0';
            ie->list = new std::vector<cfgentry*>();
            ie->key = strclone(keyp + 1);
            e = ie;
            r->list->push_back(ie);
        } else {
            keyp = strtok_r(buf, "=", &valuep);
            if (keyp && valuep) {
                char *vn = strlrtrim(keyp);
                ie->key = strclone(vn);
                ie->value = strclone(strlrtrim(valuep));
                e->list->push_back(ie);
            } else {
                delete ie;
                return lineno;
            }
        }
    }
    return 0;
}

int sds_config_parse_file(SdsConfig *cfg, const char *path)
{
    FILE *fp = fopen(path, "r");
    if (!fp)
        return -1;

    cfgentry *root = cfgentry_new();
    root->list = new std::vector<cfgentry*>();
    if (parse(fp, root)) {
        cfgentry_free(root);
        return -2;
    }

    cfg->work_dir_path = lookup_string(root, WORK_DIR_PATH, nullptr);
    if (!cfg->work_dir_path) {
        cfgentry_free(root);
        return -3;
    }

    cfg->log_to_file = lookup_bool(root, LOG_TO_FILE, 0);
    cfg->log_file_name = lookup_string(root, LOG_FILE_NAME, DEFAULT_LOG_FILE_NAME);

    cfg->db_cache_sz = lookup_int(root, DB_CACHE_SIZE, 512);
    cfg->allow_result_invalidate = lookup_bool(root, ALLOW_RESULTS_INVALIDATION, 1);
    cfg->force_tor_proxy = lookup_bool(root, FORCE_TOR_PROXY, 0);
    cfg->tor_socks5_addr = lookup_string(root, TOR_SOCKS5_ADDR, "127.0.0.1");
    cfg->tor_socks5_port = lookup_int(root, TOR_SOCKS5_PORT, 9050);

    // Tor control port
    cfg->tor_control_addr = lookup_string(root, TOR_CONTROL_ADDR, nullptr);
    if (cfg->tor_control_addr) {
        cfg->tor_control_port = lookup_int(root, TOR_CONTROL_PORT, 9051);
        cfg->tor_auth_cookie = lookup_bool(root, TOR_CONTROL_AUTH_COOKIE, 0);
        cfg->tor_password = nullptr;
        if (cfg->tor_auth_cookie) {
            cfg->tor_cookie_file_path = lookup_string(root, TOR_CONTROL_COOKIE_FILE, nullptr);
        } else {
            cfg->tor_password = lookup_string(root, TOR_CONTROL_PASSWORD, nullptr);
        }
    }

    // I2P Sam
    cfg->i2p_sam_addr = lookup_string(root, I2P_SAM_ADDR, nullptr);
    if (cfg->i2p_sam_addr) {
        cfg->i2p_sam_port = lookup_int(root, I2P_SAM_PORT, 0);
        cfg->i2p_sam_user = lookup_string(root, I2P_SAM_USER, nullptr);
        cfg->i2p_sam_password = lookup_string(root, I2P_SAM_PASSWORD, nullptr);
    }

    cfgentry *e = lookup(root, PEERS);
    if (e) {
        lookup_strings(e, cfg->known_peers, ADDRESS);
    }

    e = lookup(root, BLACKLIST_PEERS);
    if (e) {
        lookup_strings(e, cfg->blacklisted_peers, ADDRESS);
    }

    cfg->web_ui_bind_addr = lookup_string(root, WEB_UI_BIND_ADDR, nullptr);

    char *hidden_service = lookup_string(root, P2P_HIDDEN_SERVICE, NONE);
    cfg->p2p_hidden_service = NO_HIDDEN_SERVICE;
    if (!strcmp(hidden_service, TOR))
        cfg->p2p_hidden_service = TOR_HIDDEN_SERVICE;
    else if (!strcmp(hidden_service, I2P))
        cfg->p2p_hidden_service = I2P_HIDDEN_SERVICE;

    delete[] hidden_service;

    cfg->p2p_server_bind_port = lookup_int(root, P2P_BIND_PORT, 4111);
    if (!cfg->p2p_hidden_service)
        cfg->p2p_server_bind_addr = lookup_string(root, P2P_BIND_ADDR, "127.0.0.1");

    std::vector<cfgentry*> entries;
    lookups(root, entries, EXTERNAL_SEARCH_ENGINE);
    for (auto it = entries.begin(); it < entries.end(); it++) {
        SearchEngineConfigs se = {
            .name = lookup_string(*it, NAME, nullptr),
            .userAgent = lookup_string(*it, USER_AGENT, nullptr),
            .searchQueryUrl = lookup_string(*it, SEARCH_QUERY_URL, nullptr),
            .resultsContainerElement = lookup_string(*it, RESULTS_CONTAINER_ELEMENT, nullptr),
            .resultContainerElement = lookup_string(*it, RESULT_CONTAINER_ELEMENT, nullptr),
            .resultUrlElement = lookup_string(*it, RESULT_URL_ELEMENT, nullptr),
            .resultUrlProperty = lookup_string(*it, RESULT_URL_PROPERTY, nullptr),
            .resultUrlIsJson = lookup_bool(*it, RESULT_URL_IS_JSON, 0),
            .resultUrlJsonProperty = lookup_string(*it, RESULT_URL_JSON_PROPERTY, nullptr),
            .resultTitleElement = lookup_string(*it, RESULT_TITLE_ELEMENT, nullptr),
            .resultTitleProperty = lookup_string(*it, RESULT_TITLE_PROPERTY, nullptr),
            .providedDataType = lookup_string(*it, PROVIDED_DATA_TYPE, nullptr),
        };
        cfg->search_engines.push_back(se);
    }

    cfgentry_free(root);
    return 0;
}

void sds_config_print(SdsConfig *cfg)
{
    printf("work_dir_path %s\n", cfg->work_dir_path);
    printf("log_file_name %s\n", cfg->log_file_name);
    printf("log_to_file %d\n", cfg->log_to_file);
    printf("db_cache_sz %d\n", cfg->db_cache_sz);
    printf("allow_result_invalidate %d\n", cfg->allow_result_invalidate);
    printf("web_ui_bind_addr %s\n", cfg->web_ui_bind_addr);
    printf("peers = \n");
    for (auto it = cfg->known_peers.begin(); it != cfg->known_peers.end(); it++)
        printf("\tpeer %s\n", *it);
    printf("blacklist_peers = \n");
    for (auto it = cfg->blacklisted_peers.begin(); it != cfg->blacklisted_peers.end(); it++)
        printf("\tpeer %s\n", *it);
    printf("force_tor %d\n", cfg->force_tor_proxy);
    printf("tor_socks5_addr %s\n", cfg->tor_socks5_addr);
    printf("tor_socks5_port %d\n", cfg->tor_socks5_port);
    printf("p2p_server_bind_addr %s\n", cfg->p2p_server_bind_addr);
    printf("p2p_server_bind_port %d\n", cfg->p2p_server_bind_port);
    printf("p2p_hidden_service %d\n", cfg->p2p_hidden_service);
    printf("tor_control_addr %s\n", cfg->tor_control_addr);
    printf("tor_control_port %d\n", cfg->tor_control_port);
    printf("tor_auth_cookie %d\n", cfg->tor_auth_cookie);
    printf("tor_cookie_file_path %s\n", cfg->tor_cookie_file_path);
    printf("i2p_sam_addr %s\n", cfg->i2p_sam_addr);
    printf("i2p_sam_port %d\n", cfg->i2p_sam_port);
    printf("i2p_sam_user %s\n", cfg->i2p_sam_user);
    printf("i2p_sam_password %s\n", cfg->i2p_sam_password);

    printf("external_search_engines = \n");
    for (auto it = cfg->search_engines.begin(); it != cfg->search_engines.end(); it++)
        printf("\tengine %s\n", it->name);
}
