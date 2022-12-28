package sds

import (
	"fmt"
	"net"
	"strings"
)

func readDaemonAnswer(conn net.Conn) string {
	buffer := make([]byte, BUFFER_SIZE)
	_, err := conn.Read(buffer)
	if err != nil {
		panic("Failed to create hidden service can't connect to tor daemon")
	}
	if strings.Contains(string(buffer), "250-OK") {
		return string(buffer)
	}
	return ""
}

func CreateHiddenService(configs SdsConfig) Peer {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", configs.TorControlPort))
	if err != nil {
		panic("Failed to create hidden service can't connect to tor daemon")
	}
	conn.Write([]byte(fmt.Sprintf("authenticate \"%s\"", configs.TorControlPassword)))
	readDaemonAnswer(conn)

	addr, err := net.ResolveTCPAddr("tcp", configs.NodeServiceBindAddress)
	if err != nil {
		panic(fmt.Sprintf("Malformed url: %s", configs.NodeServiceBindAddress))
	}
	conn.Write([]byte(fmt.Sprintf("ADD_ONION NEW:BEST Port=%d,%s", addr.Port, configs.NodeServiceBindAddress)))

	return Peer{}
}
