package sds

import (
	"strings"

	"gopkg.in/ini.v1"
)

const SERVICE_RPC_PORT = "service_rpc_port"
const SEARCH_DATABASE_PATH = "search_database_path"
const PEERS_DATABASE_PATH = "peers_database_path"

type SdsConfig struct {
	searchDatabasePath      string
	peersDatabasePath       string
	WebServiceBindAddress   string
	KnownPeers              []Peer
	NodeServiceEnabled      bool
	AutoCreateHiddenService bool
	TorControlPort          int
	TorControlPassword      string
	NodeServiceBindAddress  string
	NodePeerInfo            Peer
	searchEngines           map[string]SearchEngine
}

func NewSdsConfig(path string) SdsConfig {
	cfg := SdsConfig{}
	cfg.fromConfigFile(path)
	return cfg
}

func (cfg *SdsConfig) fromConfigFile(path string) {
	iniData, err := ini.Load(path)

	if err != nil {

	}

	cfg.searchDatabasePath = iniData.Section(ini.DEFAULT_SECTION).Key(SEARCH_DATABASE_PATH).String()
	cfg.peersDatabasePath = iniData.Section(ini.DEFAULT_SECTION).Key(PEERS_DATABASE_PATH).String()

	cfg.WebServiceBindAddress = iniData.Section(ini.DEFAULT_SECTION).Key("web_service_bind_address").String()

	nodeServiceSection := iniData.Section("node_service")
	cfg.NodeServiceEnabled, err = nodeServiceSection.Key("enabled").Bool()
	if err != nil {
		cfg.NodeServiceEnabled = true
	}
	if cfg.NodeServiceEnabled {
		cfg.AutoCreateHiddenService, err = nodeServiceSection.Key("create_hidden_service").Bool()
		if err != nil {
			cfg.AutoCreateHiddenService = false
		}
		if cfg.AutoCreateHiddenService {
			cfg.TorControlPort, err = nodeServiceSection.Key("tor_control_port").Int()
			if err != nil {
				cfg.TorControlPort = 9051
			}
			cfg.TorControlPassword = nodeServiceSection.Key("tor_control_auth_password").String()
			cfg.NodeServiceBindAddress = nodeServiceSection.Key("bind_address").String()
		} else {
			cfg.NodePeerInfo = parsePeer(nodeServiceSection, "bind_address")
		}
	}

	peerNames := iniData.Section(ini.DEFAULT_SECTION).Key("known_peers").Strings(",")

	cfg.KnownPeers = make([]Peer, 0)
	for _, peerName := range peerNames {
		peer := iniData.Section(strings.Trim(peerName, " "))
		cfg.KnownPeers = append(cfg.KnownPeers, parsePeer(peer, "address"))
	}

	engineNames := iniData.Section(ini.DEFAULT_SECTION).Key("external_search_engines").Strings(",")

	cfg.searchEngines = make(map[string]SearchEngine)
	for _, engineName := range engineNames {
		engineKey := strings.Trim(engineName, " ")
		engine := iniData.Section(engineName)
		cfg.searchEngines[engineKey] = SearchEngine{
			name:                    engine.Key("name").String(),
			userAgent:               engine.Key("user_agent").String(),
			searchQueryUrl:          engine.Key("search_query_url").String(),
			resultsContainerElement: engine.Key("results_container_element").String(),
			resultContainerElement:  engine.Key("result_container_element").String(),
			resultUrlElement:        engine.Key("result_url_element").String(),
			resultUrlProperty:       engine.Key("result_url_property").String(),
			resultTitleElement:      engine.Key("result_title_element").String(),
			resultTitleProperty:     engine.Key("result_title_property").String(),
		}

	}

}

func parsePeer(sec *ini.Section, addressKey string) Peer {
	proxyType := stringToProxyTyeInt(sec.Key("proxy_type").String())
	proxyAddr := ""
	if proxyType != NONE_PROXY_TYPE {
		proxyAddr = sec.Key("proxy_address").String()
	}
	return Peer{
		Address:      sec.Key(addressKey).String(),
		ProxyType:    proxyType,
		ProxyAddress: proxyAddr,
	}
}

/* Utils */
func stringToProxyTyeInt(proxyType string) int {
	switch strings.ToUpper(proxyType) {
	case "SOCKS5":
		return SOCKS_5_PROXY_TYPE
	case "NONE":
	default:
		return NONE_PROXY_TYPE
	}
	return NONE_PROXY_TYPE
}
