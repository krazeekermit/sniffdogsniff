package hiddenservice

import (
	"net"
)

type NetProtocol interface {
	Listen() (net.Listener, error)

	GetAddressString() string
	Close() error
}
