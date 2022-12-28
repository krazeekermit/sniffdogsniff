package sds

import (
	"fmt"
	"net"
)

func CreateHiddenService(configs SdsConfig) Peer {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", configs.TorControlPort))
	if err != nil {
		panic("Failed to create hidden service can't connect to tor daemon")
	}
	conn.Write([]byte(fmt.Sprintf("authenticate \"%s\"", configs.TorControlPassword)))
	buffer := make([]byte, BUFFER_SIZE)
	_, err = conn.Read(buffer)
	if err != nil {
		panic("Tor control port authentication failed")
	}
	return Peer{}
}
