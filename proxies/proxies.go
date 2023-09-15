package proxies

import (
	"net"
	"strings"
	"time"

	"github.com/sniffdogsniff/logging"
	"golang.org/x/net/proxy"
)

const ONION_SUFFIX string = "onion"
const I2P_SUFFIX string = "i2p"

type ProxyType int

const (
	NONE_PROXY_TYPE        ProxyType = -1
	TOR_SOCKS_5_PROXY_TYPE ProxyType = 0
	I2P_SOCKS_5_PROXY_TYPE ProxyType = 1
)

type ProxySettings struct {
	I2pSocks5Addr string
	TorSocks5Addr string
	ForceTor      bool
}

func (ps ProxySettings) TypeByAddr(address string) (ProxyType, error) {
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return NONE_PROXY_TYPE, err
	}
	split := strings.Split(addr.IP.String(), ".")
	suffix := split[len(split)-1]
	if suffix == ONION_SUFFIX {
		return TOR_SOCKS_5_PROXY_TYPE, nil
	}

	return NONE_PROXY_TYPE, nil
}

func (ps ProxySettings) GetProxyAddress(ptype ProxyType) string {
	switch ptype {
	case TOR_SOCKS_5_PROXY_TYPE:
		return ps.TorSocks5Addr
	case I2P_SOCKS_5_PROXY_TYPE:
		return ps.I2pSocks5Addr
	}
	return ""
}

func (ps ProxySettings) NewConnection(address string) (net.Conn, error) {
	ptype, err := ps.TypeByAddr(address)
	if err != nil {
		logging.LogTrace("ProxyError::", err)
		return nil, err
	}
	if ps.ForceTor {
		ptype = TOR_SOCKS_5_PROXY_TYPE
	}

	if ptype == NONE_PROXY_TYPE {
		dialer := net.Dialer{Timeout: 10 * time.Second}
		return dialer.Dial("tcp", address)
	} else {
		dialer, err := proxy.SOCKS5("tcp", ps.GetProxyAddress(ptype), nil, &net.Dialer{
			Timeout:   60 * time.Second,
			KeepAlive: 30 * time.Second,
		})
		if err != nil {
			logging.LogError(err.Error())
			return nil, err
		}
		return dialer.Dial("tcp", address)
	}
}

func StringToProxyTypeInt(proxyType string) ProxyType {
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
