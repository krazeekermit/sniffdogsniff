package hiddenservice

import "net"

type IP4TCPProto struct {
	BindAddress string
}

func (ip4 *IP4TCPProto) Listen() (net.Listener, error) {
	return net.Listen("tcp", ip4.BindAddress)
}

func (ip4 *IP4TCPProto) GetAddressString() string {
	return ip4.BindAddress
}

func (ip4 *IP4TCPProto) Close() error {
	return nil
}
