package sds

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sniffdogsniff/hiddenservice"
	"github.com/sniffdogsniff/proxies"
	"github.com/sniffdogsniff/util/logging"
	"gopkg.in/ini.v1"
)

const MAX_RAM_DB_SIZE = 268435456 // 256 MB

const (
	SERVICE_RPC_PORT                   = "service_rpc_port"
	WORK_DIR_PATH                      = "work_dir_path"
	ALLOW_RESULTS_INVALIDATION         = "allow_results_invalidation"
	SEARCH_DATABASE_MAX_RAM_CACHE_SIZE = "search_database_max_ram_cache_size"
	KNOWN_PEERS                        = "known_peers"
	ADDRESS                            = "address"
	EXTERNAL_SEARCH_ENGINES            = "external_search_engines"
	NAME                               = "name"
	USER_AGENT                         = "user_agent"
	SEARCH_QUERY_URL                   = "search_query_url"
	RESULTS_CONTAINER_ELEMENT          = "results_container_element"
	RESULT_CONTAINER_ELEMENT           = "result_container_element"
	RESULT_URL_ELEMENT                 = "result_url_element"
	RESULT_URL_PROPERTY                = "result_url_property"
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
	CREATE_HIDDEN_SERVICE              = "create_hidden_service"
	TOR_CONTROL_PORT                   = "tor_control_port"
	TOR_CONTROL_AUTH_PASSWORD          = "tor_control_auth_password"
	I2P_SAM_PORT                       = "i2p_sam_port"
	I2P_SAM_USER                       = "i2p_sam_port"
	I2P_SAM_PASSWORD                   = "i2p_sam_password"
)

func panicNoKey(key string) {
	panic(fmt.Sprint("Config file parse error: required key: ", key))
}

func panicNoSection(key string) {
	panic(fmt.Sprint("Config file parse error: required section: ", key))
}

func parsePeer(sec *ini.Section, addressKey string) Peer {
	if sec.HasKey(addressKey) {
		return Peer{
			Address: sec.Key(addressKey).String(),
		}
	} else {
		panicNoKey(addressKey)
		return Peer{}
	}
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

type NodeServiceSettings struct {
	Enabled             bool
	CreateHiddenService bool
	OnionService        bool
	BindAddress         string
}

type SdsConfig struct {
	workDirPath              string
	searchDBMaxCacheSize     int
	AllowResultsInvalidation bool
	WebServiceBindAddress    string
	KnownPeers               []Peer
	proxySettings            proxies.ProxySettings
	ServiceSettings          NodeServiceSettings
	OnionServiceSettings     hiddenservice.OnionService
	searchEngines            map[string]SearchEngine
}

func NewSdsConfig(path string) SdsConfig {
	cfg := SdsConfig{}
	iniData, err := ini.Load(path)

	if err != nil {
		panic(err.Error())
	}

	defaultSection := iniData.Section(ini.DefaultSection)
	if defaultSection.HasKey(WORK_DIR_PATH) {
		cfg.workDirPath = defaultSection.Key(WORK_DIR_PATH).String()
	} else {
		panicNoKey(WORK_DIR_PATH)
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
	if defaultSection.HasKey(KNOWN_PEERS) {
		peerNames := iniData.Section(ini.DefaultSection).Key(KNOWN_PEERS).Strings(",")

		cfg.KnownPeers = make([]Peer, 0)
		for _, peerName := range peerNames {
			peer := iniData.Section(strings.Trim(peerName, " "))
			cfg.KnownPeers = append(cfg.KnownPeers, parsePeer(peer, ADDRESS))
		}
	}
	if defaultSection.HasKey(EXTERNAL_SEARCH_ENGINES) {
		engineNames := iniData.Section(ini.DefaultSection).Key(EXTERNAL_SEARCH_ENGINES).Strings(",")

		cfg.searchEngines = make(map[string]SearchEngine)
		for _, engineName := range engineNames {
			engineKey := strings.Trim(engineName, " ")
			engine := iniData.Section(engineName)
			cfg.searchEngines[engineKey] = SearchEngine{
				name:                    engine.Key(NAME).String(),
				userAgent:               engine.Key(USER_AGENT).String(),
				searchQueryUrl:          engine.Key(SEARCH_QUERY_URL).String(),
				resultsContainerElement: engine.Key(RESULTS_CONTAINER_ELEMENT).String(),
				resultContainerElement:  engine.Key(RESULT_CONTAINER_ELEMENT).String(),
				resultUrlElement:        engine.Key(RESULT_URL_ELEMENT).String(),
				resultUrlProperty:       engine.Key(RESULT_URL_PROPERTY).String(),
				resultTitleElement:      engine.Key(RESULT_TITLE_ELEMENT).String(),
				resultTitleProperty:     engine.Key(RESULT_TITLE_PROPERTY).String(),
				providedDataType:        StrToDataType(engine.Key(PROVIDED_DATA_TYPE).String()),
			}
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
			cfg.proxySettings.ForceTor, err = proxySettingsSection.Key(FORCE_TOR_PROXY).Bool()
			if err != nil {
				cfg.proxySettings.ForceTor = false
			}
		} else {
			cfg.proxySettings.ForceTor = false
		}
		if proxySettingsSection.HasKey(TOR_SOCKS5_PROXY) {
			cfg.proxySettings.TorSocks5Addr = proxySettingsSection.Key(TOR_SOCKS5_PROXY).String()
		} else {
			cfg.proxySettings.TorSocks5Addr = "127.0.0.1:9050"
		}
		if proxySettingsSection.HasKey(I2P_SOCKS5_PROXY) {
			cfg.proxySettings.I2pSocks5Addr = proxySettingsSection.Key(I2P_SOCKS5_PROXY).String()
		} else {
			cfg.proxySettings.I2pSocks5Addr = "127.0.0.1:4447"
		}
	} else {
		panicNoSection(PROXY_SETTINGS)
	}

	if iniData.HasSection(NODE_SERVICE) {
		nodeServiceSection := iniData.Section(NODE_SERVICE)
		if nodeServiceSection.HasKey(ENABLED) {
			cfg.ServiceSettings.Enabled, err = nodeServiceSection.Key(ENABLED).Bool()
			if err != nil {
				cfg.ServiceSettings.Enabled = true
			}
		} else {
			cfg.ServiceSettings.Enabled = true
		}
		if cfg.ServiceSettings.Enabled {
			if nodeServiceSection.HasKey(CREATE_HIDDEN_SERVICE) {
				cfg.ServiceSettings.CreateHiddenService, err = nodeServiceSection.Key(CREATE_HIDDEN_SERVICE).Bool()
				if err != nil {
					cfg.ServiceSettings.CreateHiddenService = false
				}
			}
			if cfg.ServiceSettings.CreateHiddenService {
				if nodeServiceSection.HasKey(TOR_CONTROL_PORT) {
					cfg.ServiceSettings.OnionService = true
					cfg.OnionServiceSettings.TorControlPort, err = nodeServiceSection.Key(TOR_CONTROL_PORT).Int()
					if err != nil {
						cfg.OnionServiceSettings.TorControlPort = 9051
					}
					if nodeServiceSection.HasKey(TOR_CONTROL_AUTH_PASSWORD) {
						cfg.OnionServiceSettings.TorControlPassword = nodeServiceSection.Key(TOR_CONTROL_AUTH_PASSWORD).String()
					}
				} else if nodeServiceSection.HasKey(I2P_SAM_PORT) {
					/* I2P hidden service not supported for now */
					// cfg.ServiceSettings.HiddenServiceSettings.IsTor = false
					// cfg.ServiceSettings.HiddenServiceSettings.SamPort, err = nodeServiceSection.Key(I2P_SAM_PORT).Int()
					// if err != nil {
					// 	cfg.ServiceSettings.HiddenServiceSettings.SamPort = 7656
					// }
					// if nodeServiceSection.HasKey(I2P_SAM_USER) {
					// 	cfg.ServiceSettings.HiddenServiceSettings.NeedAuth = true
					// 	cfg.ServiceSettings.HiddenServiceSettings.SamUser = nodeServiceSection.Key(I2P_SAM_USER).String()
					// 	if nodeServiceSection.HasKey(I2P_SAM_PASSWORD) {
					// 		cfg.ServiceSettings.HiddenServiceSettings.SamPassword = nodeServiceSection.Key(I2P_SAM_PASSWORD).String()
					// 	} else {
					// 		panicNoKey(I2P_SAM_PASSWORD)
					// 	}
					// } else {
					// 	cfg.ServiceSettings.HiddenServiceSettings.NeedAuth = false
					// }
				}
			}
			if nodeServiceSection.HasKey(BIND_ADDRESS) {
				cfg.ServiceSettings.BindAddress = nodeServiceSection.Key(BIND_ADDRESS).String()
			} else {
				cfg.ServiceSettings.BindAddress = "127.0.0.1:4222"
			}
		}
	} else {
		panicNoSection(NODE_SERVICE)
	}
	return cfg

}
