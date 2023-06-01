package sds

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sniffdogsniff/util/logging"
	"gopkg.in/ini.v1"
)

const MAX_RAM_DB_SIZE = 268435456 // 256 MB

const (
	SERVICE_RPC_PORT                   = "service_rpc_port"
	WORK_DIR_PATH                      = "work_dir_path"
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
	WEB_UI                             = "web_ui"
	BIND_ADDRESS                       = "bind_address"
	PROXY_SETTINGS                     = "proxy_settings"
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

type ProxySettings struct {
	i2pSocks5Addr string
	torSocks5Addr string
}

func (ps ProxySettings) AddrByType(proxyType int) string {
	switch proxyType {
	case I2P_SOCKS_5_PROXY_TYPE:
		return ps.i2pSocks5Addr
	case TOR_SOCKS_5_PROXY_TYPE:
		return ps.torSocks5Addr
	}
	return ""
}

func parsePeer(sec *ini.Section, addressKey string) Peer {
	if sec.HasKey(addressKey) {
		proxyType := stringToProxyTypeInt(sec.Key("proxy_type").String())
		return Peer{
			Address:   sec.Key(addressKey).String(),
			ProxyType: proxyType,
		}
	} else {
		panicNoKey(addressKey)
		return Peer{}
	}
}

func stringToProxyTypeInt(proxyType string) int {
	switch strings.ToUpper(proxyType) {
	case "TOR":
		return TOR_SOCKS_5_PROXY_TYPE
	case "I2P":
		return I2P_SOCKS_5_PROXY_TYPE
	case "NONE":
		return NONE_PROXY_TYPE
	default:
		return NONE_PROXY_TYPE
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

type NodeServiceSettings struct {
	Enabled               bool
	CreateHiddenService   bool
	HiddenServiceSettings HiddenService_settings
	PeerInfo              Peer
}

type SdsConfig struct {
	workDirPath           string
	searchDBMaxCacheSize  int
	WebServiceBindAddress string
	KnownPeers            []Peer
	proxySettings         ProxySettings
	ServiceSettings       NodeServiceSettings
	searchEngines         map[string]SearchEngine
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
		if proxySettingsSection.HasKey(TOR_SOCKS5_PROXY) {
			cfg.proxySettings.torSocks5Addr = proxySettingsSection.Key(TOR_SOCKS5_PROXY).String()
		} else {
			cfg.proxySettings.torSocks5Addr = "127.0.0.1:9050"
		}
		if proxySettingsSection.HasKey(I2P_SOCKS5_PROXY) {
			cfg.proxySettings.i2pSocks5Addr = proxySettingsSection.Key(I2P_SOCKS5_PROXY).String()
		} else {
			cfg.proxySettings.i2pSocks5Addr = "127.0.0.1:4447"
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
					cfg.ServiceSettings.HiddenServiceSettings.IsTor = true
					cfg.ServiceSettings.HiddenServiceSettings.TorControlPort, err = nodeServiceSection.Key(TOR_CONTROL_PORT).Int()
					if err != nil {
						cfg.ServiceSettings.HiddenServiceSettings.TorControlPort = 9051
					}
					if nodeServiceSection.HasKey(TOR_CONTROL_AUTH_PASSWORD) {
						cfg.ServiceSettings.HiddenServiceSettings.TorControlPassword = nodeServiceSection.Key(TOR_CONTROL_AUTH_PASSWORD).String()
					}
				} else if nodeServiceSection.HasKey(I2P_SAM_PORT) {
					cfg.ServiceSettings.HiddenServiceSettings.IsTor = false
					cfg.ServiceSettings.HiddenServiceSettings.SamPort, err = nodeServiceSection.Key(I2P_SAM_PORT).Int()
					if err != nil {
						cfg.ServiceSettings.HiddenServiceSettings.SamPort = 7656
					}
					if nodeServiceSection.HasKey(I2P_SAM_USER) {
						cfg.ServiceSettings.HiddenServiceSettings.NeedAuth = true
						cfg.ServiceSettings.HiddenServiceSettings.SamUser = nodeServiceSection.Key(I2P_SAM_USER).String()
						if nodeServiceSection.HasKey(I2P_SAM_PASSWORD) {
							cfg.ServiceSettings.HiddenServiceSettings.SamPassword = nodeServiceSection.Key(I2P_SAM_PASSWORD).String()
						} else {
							panicNoKey(I2P_SAM_PASSWORD)
						}
					} else {
						cfg.ServiceSettings.HiddenServiceSettings.NeedAuth = false
					}
				}
			}
			cfg.ServiceSettings.PeerInfo = parsePeer(nodeServiceSection, BIND_ADDRESS)
		}
	} else {
		panicNoSection(NODE_SERVICE)
	}
	return cfg

}
