package hiddenservice

import (
	"net"
)

type NetProto interface {
	Listen() (net.Listener, error)

	GetAddressString() string
	Close()
}
