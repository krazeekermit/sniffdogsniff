package hiddenservice

import (
	"fmt"
	"net"
)

type I2PProto struct {
	NeedAuth    bool
	SamAPIPort  int
	SamUser     string
	SamPassword string
}

// goSam is not working for our purposes so we need to implement it ourselves
// not fully implemented do not use it!
func (i2p *I2PProto) Listen() (net.Listener, error) {
	_, err := net.Dial("tcp", fmt.Sprintf(":%d", i2p.SamAPIPort))
	if err != nil {
		panic("failed to connect to the i2p daemon")
	}

	return nil, nil
}

func (i2p *I2PProto) GetAddressString() string {
	return ""
}

func (i2p *I2PProto) Close() {}
