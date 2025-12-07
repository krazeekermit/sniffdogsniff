#include "sdsconfigfile.h"

#include <cstring>

// Old Config keys
// #define SERVICE_RPC_PORT                    "service_rpc_port"
// #define WORK_DIR_PATH                       "work_dir_path"
// #define LOG_FILE_NAME                       "log_file_name"
// #define LOG_TO_FILE                         "log_to_file"
// #define ALLOW_RESULTS_INVALIDATION          "allow_results_invalidation"
// #define DB_CACHE_SIZE                       "db_cache_size"
// #define PEERS                               "peers"
// #define BLACKLIST_PEERS                     "blacklist_peers"
// #define ID                                  "id"
// #define ADDRESS                             "address"
// #define EXTERNAL_SEARCH_ENGINE              "external_search_engine"
// #define NAME                                "name"
// #define USER_AGENT                          "user_agent"
// #define SEARCH_QUERY_URL                    "search_query_url"
// #define RESULTS_CONTAINER_ELEMENT           "results_container_element"
// #define RESULT_CONTAINER_ELEMENT            "result_container_element"
// #define RESULT_URL_ELEMENT                  "result_url_element"
// #define RESULT_URL_PROPERTY                 "result_url_property"
// #define RESULT_URL_IS_JSON                  "result_url_is_json"
// #define RESULT_URL_JSON_PROPERTY            "result_url_json_property"
// #define RESULT_TITLE_ELEMENT                "result_title_element"
// #define RESULT_TITLE_PROPERTY               "result_title_property"
// #define PROVIDED_DATA_TYPE                  "provided_data_type"
// #define WEB_UI_BIND_ADDR                    "web_ui_bind_addr"
// #define WEB_UI_BIND_PORT                    "web_ui_bind_port"
// #define PROXY_SETTINGS                      "proxy_settings"
// #define FORCE_TOR_PROXY                     "force_tor_proxy"
// #define TOR_SOCKS5_ADDR                     "tor_socks5_addr"
// #define TOR_SOCKS5_PORT                     "tor_socks5_port"
// #define P2P_HIDDEN_SERVICE                  "p2p_hidden_service"
// #define NONE                                "none"
// #define TOR                                 "tor"
// #define TOR_CONTROL_ADDR                    "tor_control_addr"
// #define TOR_CONTROL_PORT                    "tor_control_port"
// #define TOR_CONTROL_PASSWORD                "tor_control_password"
// #define TOR_CONTROL_AUTH_COOKIE             "tor_control_auth_cookie"
// #define TOR_CONTROL_COOKIE_FILE             "tor_cookie_file"
// #define I2P                                 "i2p"
// #define I2P_SAM_ADDR                        "i2p_sam_addr"
// #define I2P_SAM_PORT                        "i2p_sam_port"
// #define I2P_SAM_USER                        "i2p_sam_user"
// #define I2P_SAM_PASSWORD                    "i2p_sam_password"
// #define P2P_BIND_ADDR                       "p2p_bind_addr"
// #define P2P_BIND_PORT                       "p2p_bind_port"
// #define STUN_SERVER_ADDR                    "stun_server_addr"
// #define STUN_SERVER_PORT                    "stun_server_port"

/*
 * SdsConfigFile::Section
 */
std::string SdsConfigFile::Section::getName() const
{
    return this->name;
}

bool SdsConfigFile::Section::hasValue(const char *key)
{
    std::string val = "";
    return this->lookupValue(key, val);
}

std::string SdsConfigFile::Section::lookupString(const char *key, const char *defaultValue)
{
    std::string val = "";
    if (this->lookupValue(key, val)) {
        return val;
    }

    return defaultValue;
}

void SdsConfigFile::Section::lookupStrings(const char *key, std::vector<std::string> &list)
{
    for (auto it = this->values.begin(); it != this->values.end(); ++it) {
        if (it->first == key) {
            list.push_back(it->second);
        }
    }
}

bool SdsConfigFile::Section::lookupBool(const char *key, bool defaultValue)
{
    std::string val = "";
    if (this->lookupValue(key, val)) {
        return val == "true" || val == "yes";
    }

    return defaultValue;
}

int SdsConfigFile::Section::lookupInt(const char *key, int defaultValue)
{
    std::string val = "";
    if (this->lookupValue(key, val)) {
        return std::stoi(val);
    }

    return defaultValue;
}

std::ostream &operator<<(std::ostream &os, const SdsConfigFile::Section *section)
{
    os << "[\n";
    for (auto it = section->values.begin(); it != section->values.end(); ++it) {
        os << it->first << "=" << it->second << ",\n";
    }
    os << "]";
    return os;
}

SdsConfigFile::Section::Section(const char *_name)
    : name(_name)
{}

bool SdsConfigFile::Section::lookupValue(const char *key, std::string &value)
{
    for (auto it = this->values.begin(); it != this->values.end(); ++it) {
        if (it->first == key) {
            value = it->second;
            return true;
        }
    }

    return false;
}

/*
 * SdsConfigFile
 */
SdsConfigFile::SdsConfigFile()
    : defaultSection(new SdsConfigFile::Section("default"))
{}

SdsConfigFile::~SdsConfigFile()
{
    delete this->defaultSection;
    for (auto it = this->sections.begin(); it != this->sections.end(); it++) {
        delete *it;
    }
}

bool SdsConfigFile::hasSection(const char *key)
{
    return this->lookupSection(key) != nullptr;
}

SdsConfigFile::Section *SdsConfigFile::lookupSection(const char *name)
{
    for (auto it = this->sections.begin(); it != this->sections.end(); ++it) {
        if ((*it)->getName() == name) {
            return *it;
        }
    }

    return nullptr;
}

void SdsConfigFile::lookupSections(const char *name, std::vector<Section *> &list)
{
    for (auto it = this->sections.begin(); it != this->sections.end(); ++it) {
        if ((*it)->getName() == name) {
            list.push_back(*it);
        }
    }
}

static char *strlrtrim(char *s)
{
    int l = strlen(s);
    char *sep = s + l - 1;
    for (; s < sep && (*s == ' ' || *s == '\t'); s++);
    while ((*sep == ' ' || *sep == '\n' || *sep == '\t'))
        *sep = '\0';

    return s;
}

void SdsConfigFile::parse(const char *path)
{
    int lineno = 0;
    char buf[512];
    char *keyp = nullptr, *valuep = nullptr, *sepp = nullptr;

    FILE *fp = fopen(path, "r");
    if (!fp) {
        throw std::runtime_error(strerror(errno));
    }

    SdsConfigFile::Section *section = this->defaultSection;

    while (fgets(buf, sizeof(buf), fp)) {
        lineno++;
        keyp = strchr(buf, '#');
        if (keyp)
            *keyp = '\0';
        strlrtrim(buf);
        if (buf[0] == '\0')
            continue;

        keyp = strchr(buf, '[');
        valuep = strchr(buf, ']');
        if (keyp && valuep) {
            *valuep = '\0';
            if (!strlen(keyp + 1)) {
                throw std::runtime_error("parse error: invalid section name at line " + std::to_string(lineno));
            }

            section = new SdsConfigFile::Section(keyp + 1);
            this->sections.push_back(section);
        } else {
            sepp = strchr(buf, '=');
            if (!sepp) {
                throw std::runtime_error("parse error: invalid syntax at line " + std::to_string(lineno));
            }

            *sepp = '\0';
            keyp = buf;
            valuep = sepp + 1;

            char *k = strlrtrim(keyp);
            if (!strlen(k)) {
                throw std::runtime_error("parse error: invalid syntax at line " + std::to_string(lineno));
            }

            char *v = strlrtrim(valuep);
            if (!strlen(v)) {
                throw std::runtime_error("parse error: empty value for key \"" + std::string(k) + "\" at line " + std::to_string(lineno));
            }

            section->values.push_back(std::make_pair(k, v));
        }
    }
}

SdsConfigFile::Section *SdsConfigFile::getDefaultSection() const
{
    return defaultSection;
}

std::ostream &operator<<(std::ostream &os, const SdsConfigFile *cfg)
{
    os << "SdsConfigFile[\n";
    os << cfg->defaultSection->getName() << "=" << cfg->defaultSection;
    if (!cfg->sections.empty()) {
        for (auto sit = cfg->sections.begin(); sit != cfg->sections.end(); ++sit) {
            os << ",\n" << (*sit)->getName() << "=" << *sit;
        }
    }

    os << "]\n";
    return os;
}
