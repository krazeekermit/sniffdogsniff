package hiddenservice

import (
	"net"
)

type NetTransport interface {
	Listen() (net.Listener, error)
	Close()
}
