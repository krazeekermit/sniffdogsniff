package proxies

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/sniffdogsniff/logging"
	"github.com/sniffdogsniff/proxies/i2p"
	"golang.org/x/net/proxy"
)

const PROXY = "proxy"

const ONION_SUFFIX string = "onion"
const I2P_SUFFIX string = "i2p"

type ProxyType int

const (
	NONE_PROXY_TYPE ProxyType = -1
	TOR_PROXY_TYPE  ProxyType = 0
	I2P_PROXY_TYPE  ProxyType = 1
)

var i2pContext i2p.I2PCtx
var i2pSamSession *i2p.I2PSamSession = nil
var torSocks5Addr string
var forceUseTor = false

func InitProxySettings(i2pCtx i2p.I2PCtx, i2pSession *i2p.I2PSamSession, torSocks5 string, forceTor bool) {
	i2pContext = i2pCtx
	i2pSamSession = i2pSession
	torSocks5Addr = torSocks5
	forceUseTor = forceTor

}

func typeByAddr(address string) (ProxyType, error) {
	addrHost, _, err := net.SplitHostPort(address)
	if err != nil {
		return NONE_PROXY_TYPE, err
	}
	split := strings.Split(addrHost, ".")
	suffix := split[len(split)-1]
	if suffix == ONION_SUFFIX {
		return TOR_PROXY_TYPE, nil
	} else if suffix == I2P_SUFFIX {

	}

	return NONE_PROXY_TYPE, nil
}

func NewConnection(address string) (net.Conn, error) {
	ptype, err := typeByAddr(address)
	if err != nil {
		logging.Debugf(PROXY, err.Error())
		return nil, err
	}
	if forceUseTor {
		ptype = TOR_PROXY_TYPE
	}

	if ptype == NONE_PROXY_TYPE {
		dialer := net.Dialer{Timeout: 10 * time.Second}
		return dialer.Dial("tcp", address)
	} else if ptype == TOR_PROXY_TYPE {
		dialer, err := proxy.SOCKS5("tcp", torSocks5Addr, nil, &net.Dialer{
			Timeout:   60 * time.Second,
			KeepAlive: 30 * time.Second,
		})
		if err != nil {
			logging.Errorf(PROXY, "%s: %s", address, err.Error())
			return nil, err
		}
		return dialer.Dial("tcp", address)
	} else if ptype == I2P_PROXY_TYPE {
		if i2pSamSession == nil {
			session, err := i2p.NewI2PSamSession_Transient(i2pContext)
			if err != nil {
				return nil, err
			}
			i2pSamSession = session
		}
		return i2pSamSession.I2PDial(address)
	}
	return nil, fmt.Errorf("proxy error: no proxy configured")
}

func StringToProxyTypeInt(proxyType string) ProxyType {
	switch strings.ToUpper(proxyType) {
	case "tor":
		return TOR_PROXY_TYPE
	case "i2p":
		return I2P_PROXY_TYPE
	case "none":
		return NONE_PROXY_TYPE
	default:
		return NONE_PROXY_TYPE
	}
}
