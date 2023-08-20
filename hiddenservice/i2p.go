package hiddenservice

import (
	"fmt"
	"log"
	"net"

	// we use an external library for i2p. i2p proto is different than tor so it needs
	// custom transports

	"github.com/eyedeekay/goSam"
)

type I2PProto struct {
	NeedAuth    bool
	SamAPIPort  int
	SamUser     string
	SamPassword string

	client *goSam.Client
}

func (i2p *I2PProto) Listen() (net.Listener, error) {
	sam, err := goSam.NewClient(fmt.Sprintf(":%d", i2p.SamAPIPort))
	if err != nil {
		return nil, err
	}
	log.Println("Client Created")
	return sam.Listen()
}

func (i2p *I2PProto) GetAddressString() string {
	return i2p.client.Base32()
}

func (i2p *I2PProto) Close() {}
