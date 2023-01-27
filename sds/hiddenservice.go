package sds

import (
	"fmt"
	"net"
	"strings"

	"gitlab.com/sniffdogsniff/util/logging"
)

const TXT_BUFFER_SIZE = 2048

func readDaemonAnswer(conn net.Conn) []string {
	recvBytes := make([]byte, 0)
	buf := make([]byte, TXT_BUFFER_SIZE)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			break
		}
		recvBytes = append(recvBytes, buf[:n]...)
		if n < TXT_BUFFER_SIZE {
			break
		}
	}
	logging.LogTrace("Received response from tor daemon:", string(recvBytes))
	if strings.Contains(string(recvBytes), "OK") {
		respLines := strings.Split(string(recvBytes), "\n")
		for idx, line := range respLines {
			respLines[idx] = strings.Trim(line, "\n\r")
		}
		return respLines
	}
	return []string{""}
}

func writeToDaemon(conn net.Conn, text string) {
	logging.LogTrace("Send command to Tor daemon:", text)
	n, err := conn.Write([]byte(text + "\n"))
	if err != nil || n < len([]byte(text)) {
		panic("Failed to create hidden service can't communicate to tor daemon")
	}
}

func connectToControlPort(port int, auth string) net.Conn {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic("Failed to create hidden service can't connect to tor daemon")
	}
	writeToDaemon(conn, fmt.Sprintf("authenticate \"%s\"", auth))
	readDaemonAnswer(conn)
	return conn
}

func CreateHiddenService(serviceSettings NodeServiceSettings) Peer {
	conn := connectToControlPort(serviceSettings.TorControlPort, serviceSettings.TorControlPassword)

	addr, err := net.ResolveTCPAddr("tcp", serviceSettings.PeerInfo.Address)
	if err != nil {
		panic(fmt.Sprintf("Malformed url: %s", serviceSettings.PeerInfo.Address))
	}

	onionAddr := ""
	writeToDaemon(conn, fmt.Sprintf("ADD_ONION NEW:BEST Port=%d,%s", addr.Port, serviceSettings.PeerInfo.Address))
	response := readDaemonAnswer(conn)
	if len(response) > 0 {
		tokens := strings.Split(response[0], "=")
		if strings.Contains(tokens[0], "ServiceID") {
			onionAddr = fmt.Sprint(tokens[1], ".onion:", addr.Port)
		}
	}

	logging.LogInfo("Hidden Service started on", onionAddr)

	conn.Close()

	return Peer{
		Address:   onionAddr,
		ProxyType: TOR_SOCKS_5_PROXY_TYPE,
	}
}

func RemoveHiddenService(serviceSettings NodeServiceSettings, p Peer) {
	conn := connectToControlPort(serviceSettings.TorControlPort, serviceSettings.TorControlPassword)
	logging.LogInfo("Removing Hidden Service", p.Address)
	writeToDaemon(conn, fmt.Sprintf("DEL_ONION %s", strings.Split(p.Address, ".")[0]))
	readDaemonAnswer(conn)
	conn.Close()
}
