package core

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sniffdogsniff/kademlia"
	"github.com/sniffdogsniff/logging"
	"github.com/sniffdogsniff/proxies"
	"github.com/sniffdogsniff/proxies/i2p"
	"github.com/sniffdogsniff/proxies/tor"
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
	PEERS                              = "peers"
	PEERS_BLACKLIST                    = "peers_blacklist"
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
	TOR_CONTROL_ADDR                   = "tor_control_addr"
	TOR_CONTROL_PASSWORD               = "tor_control_password"
	TOR_CONTROL_AUTH_COOKIE            = "tor_control_auth_cookie"
	I2P                                = "i2p"
	I2P_SAM_ADDR                       = "i2p_sam_addr"
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
		logging.Warnf("config", "Cannot parse db cache size")
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
	PeersBlacklist           map[kademlia.KadId]string
	TorContext               tor.TorCtx
	ForceTor                 bool
	TorSocks5Address         string
	I2pContext               i2p.I2PCtx
	P2PServerEnabled         bool
	P2PServerBindAddress     string
	P2PBindPort              int
	P2pHiddenService         proxies.ProxyType
	searchEngines            map[string]SearchEngine
}

func NewSdsConfig(path string) SdsConfig {
	cfg := SdsConfig{}
	iniData, err := ini.LoadSources(ini.LoadOptions{
		AllowNonUniqueSections: true,
		AllowShadows:           true,
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

	if defaultSection.HasKey(FORCE_TOR_PROXY) {
		cfg.ForceTor, err = defaultSection.Key(FORCE_TOR_PROXY).Bool()
		if err != nil {
			cfg.ForceTor = false
		}
	} else {
		cfg.ForceTor = false
	}
	if defaultSection.HasKey(TOR_SOCKS5_PROXY) {
		cfg.TorSocks5Address = defaultSection.Key(TOR_SOCKS5_PROXY).String()
	} else {
		cfg.TorSocks5Address = "127.0.0.1:9050"
	}
	if defaultSection.HasKey(I2P_SOCKS5_PROXY) {
		cfg.TorSocks5Address = defaultSection.Key(I2P_SOCKS5_PROXY).String()
	} else {
		cfg.TorSocks5Address = "127.0.0.1:4447"
	}
	if defaultSection.HasKey(TOR_CONTROL_ADDR) {
		cfg.TorContext.TorControlAddr = defaultSection.Key(TOR_CONTROL_ADDR).String()
		if defaultSection.HasKey(TOR_CONTROL_AUTH_COOKIE) {
			cfg.TorContext.TorCookieAuth = defaultSection.Key(TOR_CONTROL_AUTH_COOKIE).MustBool(false)
		} else if defaultSection.HasKey(TOR_CONTROL_PASSWORD) {
			cfg.TorContext.TorControlPassword = defaultSection.Key(TOR_CONTROL_PASSWORD).String()
			cfg.TorContext.TorCookieAuth = false
		} else {
			cfg.TorContext.TorCookieAuth = true
		}
	}
	if defaultSection.HasKey(I2P_SAM_ADDR) {
		cfg.I2pContext.SamAddr = defaultSection.Key(I2P_SAM_ADDR).String()
		if defaultSection.HasKey(I2P_SAM_USER) {
			cfg.I2pContext.NeedAuth = true
			cfg.I2pContext.SamUser = defaultSection.Key(I2P_SAM_USER).String()
			if defaultSection.HasKey(I2P_SAM_PASSWORD) {
				cfg.I2pContext.SamPassword = defaultSection.Key(I2P_SAM_PASSWORD).String()
			} else {
				panicNoKey(I2P_SAM_PASSWORD)
			}
		} else {
			cfg.I2pContext.NeedAuth = false
		}
	}

	cfg.KnownPeers = make(map[kademlia.KadId]string)
	if iniData.HasSection(PEERS) {
		pSec := iniData.Section(PEERS)
		if pSec.HasKey(ADDRESS) {
			for _, addr := range pSec.Key(ADDRESS).ValueWithShadows() {
				cfg.KnownPeers[kademlia.NewKadIdFromAddrStr(addr)] = addr
			}
		} else {
			panicNoKey(ADDRESS)
		}
	} else {
		panicNoSection(PEERS)
	}

	cfg.PeersBlacklist = make(map[kademlia.KadId]string)
	if iniData.HasSection(PEERS_BLACKLIST) {
		pSec := iniData.Section(PEERS_BLACKLIST)
		if pSec.HasKey(ADDRESS) {
			for _, addr := range pSec.Key(ADDRESS).ValueWithShadows() {
				cfg.PeersBlacklist[kademlia.NewKadIdFromAddrStr(addr)] = addr
			}
		}
	}

	cfg.searchEngines = make(map[string]SearchEngine)
	enginesSections, err := iniData.SectionsByName(EXTERNAL_SEARCH_ENGINE)
	if err != nil {
		logging.Warnf("config", "Failed to parse config file: external search engines")
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
				cfg.P2pHiddenService =
					proxies.StringToProxyTypeInt(nodeServiceSection.Key(HIDDEN_SERVICE).String())
			} else {
				cfg.P2pHiddenService = proxies.NONE_PROXY_TYPE
			}

			if cfg.P2pHiddenService == proxies.NONE_PROXY_TYPE {
				if nodeServiceSection.HasKey(BIND_ADDRESS) {
					cfg.P2PServerBindAddress = nodeServiceSection.Key(BIND_ADDRESS).String()
				} else {
					cfg.P2PServerBindAddress = "127.0.0.1:4222"
				}
			} else if cfg.P2pHiddenService == proxies.TOR_PROXY_TYPE {
				cfg.P2PBindPort = nodeServiceSection.Key(BIND_PORT).MustInt(4222)
			}
		}
	} else {
		panicNoSection(NODE_SERVICE)
	}
	return cfg

}
