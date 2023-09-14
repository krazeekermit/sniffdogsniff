package core

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/sniffdogsniff/hiddenservice"
	"github.com/sniffdogsniff/kademlia"
	"github.com/sniffdogsniff/logging"
	"github.com/sniffdogsniff/proxies"
	"gopkg.in/ini.v1"
)

const MAX_RAM_DB_SIZE = 268435456 // 256 MB
const DEFAULT_BIND_PORT = 4222

const (
	DEFAULT_LOG_FILE_NAME = "sds.log"
)

const (
	SERVICE_RPC_PORT                   = "service_rpc_port"
	WORK_DIR_PATH                      = "work_dir_path"
	LOG_FILE_NAME                      = "log_file_name"
	LOG_TO_FILE                        = "log_to_file"
	ALLOW_RESULTS_INVALIDATION         = "allow_results_invalidation"
	SEARCH_DATABASE_MAX_RAM_CACHE_SIZE = "search_database_max_ram_cache_size"
	PEER                               = "peer"
	ID                                 = "id"
	ADDRESS                            = "address"
	EXTERNAL_SEARCH_ENGINE             = "external_search_engine"
	NAME                               = "name"
	USER_AGENT                         = "user_agent"
	SEARCH_QUERY_URL                   = "search_query_url"
	RESULTS_CONTAINER_ELEMENT          = "results_container_element"
	RESULT_CONTAINER_ELEMENT           = "result_container_element"
	RESULT_URL_ELEMENT                 = "result_url_element"
	RESULT_URL_PROPERTY                = "result_url_property"
	RESULT_URL_IS_JSON                 = "result_url_is_json"
	RESULT_URL_JSON_PROPERTY           = "result_url_json_property"
	RESULT_TITLE_ELEMENT               = "result_title_element"
	RESULT_TITLE_PROPERTY              = "result_title_property"
	PROVIDED_DATA_TYPE                 = "provided_data_type"
	WEB_UI                             = "web_ui"
	BIND_ADDRESS                       = "bind_address"
	PROXY_SETTINGS                     = "proxy_settings"
	FORCE_TOR_PROXY                    = "force_tor_proxy"
	I2P_SOCKS5_PROXY                   = "i2p_socks5_proxy"
	TOR_SOCKS5_PROXY                   = "tor_socks5_proxy"
	NODE_SERVICE                       = "node_service"
	ENABLED                            = "enabled"
	HIDDEN_SERVICE                     = "hidden_service"
	TOR                                = "tor"
	TOR_CONTROL_PORT                   = "tor_control_port"
	TOR_CONTROL_AUTH_PASSWORD          = "tor_control_auth_password"
	TOR_CONTROL_AUTH_COOKIE            = "tor_control_auth_cookie"
	I2P                                = "i2p"
	I2P_SAM_PORT                       = "i2p_sam_port"
	I2P_SAM_USER                       = "i2p_sam_user"
	I2P_SAM_PASSWORD                   = "i2p_sam_password"
	BIND_PORT                          = "bind_port"
)

func panicNoKey(key string) {
	panic(fmt.Sprint("Config file parse error: required key: ", key))
}

func panicNoSection(key string) {
	panic(fmt.Sprint("Config file parse error: required section: ", key))
}

func stringToByteSize(text string) int {
	cleanStr := strings.Trim(text, " ")
	size, err := strconv.Atoi(cleanStr[0 : len(cleanStr)-2])
	if err != nil {
		logging.LogWarn("Cannot parse db cache size")
		return MAX_RAM_DB_SIZE
	}
	if strings.HasSuffix(cleanStr, "G") { // Gigs
		return size * 1024 * 1024 * 1024
	} else if strings.HasSuffix(cleanStr, "M") { // Megs
		return size * 1024 * 1024
	} else if strings.HasSuffix(cleanStr, "K") { // Kilos
		return size * 1024
	} else { // Bytes
		return size
	}
}

func StrToDataType(token string) ResultDataType {
	switch token {
	case "images":
		return IMAGE_DATA_TYPE
	case "videos":
		return VIDEO_DATA_TYPE
	case "links":
		return LINK_DATA_TYPE
	default:
		return LINK_DATA_TYPE
	}
}

type SdsConfig struct {
	WorkDirPath              string
	LogFileName              string
	LogToFile                bool
	searchDBMaxCacheSize     int
	AllowResultsInvalidation bool
	WebServiceBindAddress    string
	KnownPeers               map[kademlia.KadId]string
	ProxySettings            proxies.ProxySettings
	P2PServerEnabled         bool
	P2PServerProto           hiddenservice.NetProtocol
	searchEngines            map[string]SearchEngine
}

func NewSdsConfig(path string) SdsConfig {
	cfg := SdsConfig{}
	iniData, err := ini.LoadSources(ini.LoadOptions{
		AllowNonUniqueSections: true,
	}, path)

	if err != nil {
		panic(err.Error())
	}

	defaultSection := iniData.Section(ini.DefaultSection)
	if defaultSection.HasKey(WORK_DIR_PATH) {
		cfg.WorkDirPath = defaultSection.Key(WORK_DIR_PATH).String()
	} else {
		panicNoKey(WORK_DIR_PATH)
	}
	if defaultSection.HasKey(LOG_TO_FILE) {
		cfg.LogToFile = defaultSection.Key(LOG_TO_FILE).MustBool(false)
	} else {
		cfg.LogToFile = false
	}
	if defaultSection.HasKey(LOG_FILE_NAME) {
		cfg.LogFileName = defaultSection.Key(LOG_TO_FILE).String()
	} else {
		cfg.LogFileName = DEFAULT_LOG_FILE_NAME
	}
	if defaultSection.HasKey(SEARCH_DATABASE_MAX_RAM_CACHE_SIZE) {
		cfg.searchDBMaxCacheSize = stringToByteSize(
			defaultSection.Key(SEARCH_DATABASE_MAX_RAM_CACHE_SIZE).String())
	} else {
		cfg.searchDBMaxCacheSize = MAX_RAM_DB_SIZE
	}
	if defaultSection.HasKey(ALLOW_RESULTS_INVALIDATION) {
		allowInvalidation, err := defaultSection.Key(ALLOW_RESULTS_INVALIDATION).Bool()
		if err != nil {
			cfg.AllowResultsInvalidation = allowInvalidation
		} else {
			cfg.AllowResultsInvalidation = false
		}
	} else {
		cfg.AllowResultsInvalidation = false
	}

	cfg.KnownPeers = make(map[kademlia.KadId]string)
	peersSections, err := iniData.SectionsByName(PEER)
	if err != nil {
		panicNoSection(PEER)
	}
	for _, pSec := range peersSections {
		if pSec.HasKey(ADDRESS) && pSec.HasKey(ID) {
			idBytez, err := hex.DecodeString(pSec.Key(ID).String())
			if err != nil {
				continue
			}
			kadId := kademlia.KadIdFromBytes(idBytez)
			cfg.KnownPeers[kadId] = pSec.Key(ADDRESS).String()
		}
	}

	cfg.searchEngines = make(map[string]SearchEngine)
	enginesSections, err := iniData.SectionsByName(EXTERNAL_SEARCH_ENGINE)
	if err != nil {
		logging.LogWarn("Failed to parse config file: external search engines")
	}
	for _, sec := range enginesSections {
		name := sec.Key(NAME).String()
		cfg.searchEngines[name] = SearchEngine{
			name:                    name,
			userAgent:               sec.Key(USER_AGENT).String(),
			searchQueryUrl:          sec.Key(SEARCH_QUERY_URL).String(),
			resultsContainerElement: sec.Key(RESULTS_CONTAINER_ELEMENT).String(),
			resultContainerElement:  sec.Key(RESULT_CONTAINER_ELEMENT).String(),
			resultUrlElement:        sec.Key(RESULT_URL_ELEMENT).String(),
			resultUrlProperty:       sec.Key(RESULT_URL_PROPERTY).String(),
			resultUrlIsJson:         sec.Key(RESULT_URL_IS_JSON).MustBool(false),
			resultUrlJsonProperty:   sec.Key(RESULT_URL_JSON_PROPERTY).String(),
			resultTitleElement:      sec.Key(RESULT_TITLE_ELEMENT).String(),
			resultTitleProperty:     sec.Key(RESULT_TITLE_PROPERTY).String(),
			providedDataType:        StrToDataType(sec.Key(PROVIDED_DATA_TYPE).String()),
		}
	}

	if iniData.HasSection(WEB_UI) {
		webUiSection := iniData.Section(WEB_UI)
		if webUiSection.HasKey(BIND_ADDRESS) {
			cfg.WebServiceBindAddress = webUiSection.Key(BIND_ADDRESS).String()
		} else {
			cfg.WebServiceBindAddress = "127.0.0.1:8081"
		}
	} else {
		panicNoSection(WEB_UI)
	}

	if iniData.HasSection(PROXY_SETTINGS) {
		proxySettingsSection := iniData.Section(PROXY_SETTINGS)
		if proxySettingsSection.HasKey(FORCE_TOR_PROXY) {
			cfg.ProxySettings.ForceTor, err = proxySettingsSection.Key(FORCE_TOR_PROXY).Bool()
			if err != nil {
				cfg.ProxySettings.ForceTor = false
			}
		} else {
			cfg.ProxySettings.ForceTor = false
		}
		if proxySettingsSection.HasKey(TOR_SOCKS5_PROXY) {
			cfg.ProxySettings.TorSocks5Addr = proxySettingsSection.Key(TOR_SOCKS5_PROXY).String()
		} else {
			cfg.ProxySettings.TorSocks5Addr = "127.0.0.1:9050"
		}
		if proxySettingsSection.HasKey(I2P_SOCKS5_PROXY) {
			cfg.ProxySettings.I2pSocks5Addr = proxySettingsSection.Key(I2P_SOCKS5_PROXY).String()
		} else {
			cfg.ProxySettings.I2pSocks5Addr = "127.0.0.1:4447"
		}
	} else {
		panicNoSection(PROXY_SETTINGS)
	}

	if iniData.HasSection(NODE_SERVICE) {
		nodeServiceSection := iniData.Section(NODE_SERVICE)
		if nodeServiceSection.HasKey(ENABLED) {
			cfg.P2PServerEnabled, err = nodeServiceSection.Key(ENABLED).Bool()
			if err != nil {
				cfg.P2PServerEnabled = true
			}
		} else {
			cfg.P2PServerEnabled = true
		}
		if cfg.P2PServerEnabled {
			if nodeServiceSection.HasKey(HIDDEN_SERVICE) {
				hiddenService := nodeServiceSection.Key(HIDDEN_SERVICE).String()
				if hiddenService == TOR {
					serviceProto := &hiddenservice.TorProto{}

					if nodeServiceSection.HasKey(TOR_CONTROL_PORT) {
						serviceProto.TorControlPort, err = nodeServiceSection.Key(TOR_CONTROL_PORT).Int()
						if err != nil {
							serviceProto.TorControlPort = 9051
						}
						if nodeServiceSection.HasKey(TOR_CONTROL_AUTH_COOKIE) {
							serviceProto.TorCookieAuth = nodeServiceSection.Key(TOR_CONTROL_AUTH_COOKIE).MustBool(false)
						} else if nodeServiceSection.HasKey(TOR_CONTROL_AUTH_PASSWORD) {
							serviceProto.TorControlPassword = nodeServiceSection.Key(TOR_CONTROL_AUTH_PASSWORD).String()
							serviceProto.TorCookieAuth = false
						} else {
							serviceProto.TorCookieAuth = true
						}
					}
					if nodeServiceSection.HasKey(BIND_PORT) {
						serviceProto.BindPort = nodeServiceSection.Key(BIND_PORT).MustInt(DEFAULT_BIND_PORT)
					} else {
						serviceProto.BindPort = DEFAULT_BIND_PORT
					}
					serviceProto.WorkDirPath = cfg.WorkDirPath
					cfg.P2PServerProto = serviceProto
				} else if hiddenService == I2P {

					if nodeServiceSection.HasKey(I2P_SAM_PORT) {
						serviceProto := &hiddenservice.I2PProto{}

						serviceProto.SamAPIPort, err = nodeServiceSection.Key(I2P_SAM_PORT).Int()
						if err != nil {
							serviceProto.SamAPIPort = 7656
						}
						if nodeServiceSection.HasKey(I2P_SAM_USER) {
							serviceProto.NeedAuth = true
							serviceProto.SamUser = nodeServiceSection.Key(I2P_SAM_USER).String()
							if nodeServiceSection.HasKey(I2P_SAM_PASSWORD) {
								serviceProto.SamPassword = nodeServiceSection.Key(I2P_SAM_PASSWORD).String()
							} else {
								panicNoKey(I2P_SAM_PASSWORD)
							}
						} else {
							serviceProto.NeedAuth = false
						}
						cfg.P2PServerProto = serviceProto
					}
				}
			} else {
				serviceProto := &hiddenservice.IP4TCPProto{}
				if nodeServiceSection.HasKey(BIND_ADDRESS) {
					serviceProto.BindAddress = nodeServiceSection.Key(BIND_ADDRESS).String()
				} else {
					serviceProto.BindAddress = "127.0.0.1:4222"
				}
				cfg.P2PServerProto = serviceProto
			}
		}
	} else {
		panicNoSection(NODE_SERVICE)
	}
	return cfg

}
