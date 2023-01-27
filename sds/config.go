package sds

import (
	"strings"

	"gopkg.in/ini.v1"
)

const SERVICE_RPC_PORT = "service_rpc_port"
const SEARCH_DATABASE_PATH = "search_database_path"
const PEERS_DATABASE_PATH = "peers_database_path"

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

type NodeServiceSettings struct {
	Enabled             bool
	CreateHiddenService bool
	TorControlPort      int
	TorControlPassword  string
	PeerInfo            Peer
}

type SdsConfig struct {
	searchDatabasePath    string
	peersDatabasePath     string
	WebServiceBindAddress string
	KnownPeers            []Peer
	proxySettings         ProxySettings
	ServiceSettings       NodeServiceSettings
	searchEngines         map[string]SearchEngine
}

func NewSdsConfig(path string) SdsConfig {
	cfg := SdsConfig{}
	cfg.fromConfigFile(path)
	return cfg
}

func (cfg *SdsConfig) fromConfigFile(path string) {
	iniData, err := ini.Load(path)

	if err != nil {
		panic("Failed to load config file!")
	}

	cfg.searchDatabasePath = iniData.Section(ini.DEFAULT_SECTION).Key(SEARCH_DATABASE_PATH).String()
	cfg.peersDatabasePath = iniData.Section(ini.DEFAULT_SECTION).Key(PEERS_DATABASE_PATH).String()

	cfg.WebServiceBindAddress = iniData.Section(ini.DEFAULT_SECTION).Key("web_service_bind_address").String()

	proxyConfigs := iniData.Section("proxy_settings")
	cfg.proxySettings = ProxySettings{
		i2pSocks5Addr: proxyConfigs.Key("i2p_socks5_proxy").String(),
		torSocks5Addr: proxyConfigs.Key("tor_socks5_proxy").String(),
	}

	nodeServiceSection := iniData.Section("node_service")
	cfg.ServiceSettings = NodeServiceSettings{}
	cfg.ServiceSettings.Enabled, err = nodeServiceSection.Key("enabled").Bool()
	if err != nil {
		cfg.ServiceSettings.Enabled = true
	}
	if cfg.ServiceSettings.Enabled {
		cfg.ServiceSettings.CreateHiddenService, err = nodeServiceSection.Key("create_hidden_service").Bool()
		if err != nil {
			cfg.ServiceSettings.CreateHiddenService = false
		}
		if cfg.ServiceSettings.CreateHiddenService {
			cfg.ServiceSettings.TorControlPort, err = nodeServiceSection.Key("tor_control_port").Int()
			if err != nil {
				cfg.ServiceSettings.TorControlPort = 9051
			}
			cfg.ServiceSettings.TorControlPassword = nodeServiceSection.Key("tor_control_auth_password").String()
		}
		cfg.ServiceSettings.PeerInfo = parsePeer(nodeServiceSection, "bind_address")
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
	proxyType := stringToProxyTypeInt(sec.Key("proxy_type").String())
	return Peer{
		Address:   sec.Key(addressKey).String(),
		ProxyType: proxyType,
	}
}

/* Utils */
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
