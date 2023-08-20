package hiddenservice

import (
	"fmt"
	"math/rand"
	"net"
	"strings"

	"github.com/sniffdogsniff/util/logging"
)

const TXT_BUFFER_SIZE = 2048
const DEFAULT_BIND_ADDRESS = "127.0.0.1"

func writeToDaemon(conn net.Conn, text string) {
	logging.LogTrace("Send command to daemon:", text)
	n, err := conn.Write([]byte(text + "\n"))
	if err != nil || n < len([]byte(text)) {
		panic("Failed to create hidden service can't communicate to daemon")
	}
}

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
	respLines := strings.Split(string(recvBytes), "\n")
	for idx, line := range respLines {
		respLines[idx] = strings.Trim(line, "\n\r")
	}
	return respLines
}

func readTorDaemonAnswer(conn net.Conn) []string {
	lines := readDaemonAnswer(conn)
	for _, l := range lines {
		if strings.Contains(l, "OK") {
			return lines
		}
	}
	return []string{}
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

// port between 2000 and 7000
func randPort() int {
	rnd := rand.Intn(7000)
	if rnd < 2000 {
		rnd += 2000
	}
	return rnd
}

type TorProto struct {
	NeedAuth           bool
	TorControlPort     int
	TorControlPassword string
	OnionAddress       string
}

func (ons *TorProto) connectAndCreateHiddenService(port int) {
	conn := connectToControlPort(ons.TorControlPort,
		ons.TorControlPassword)

	writeToDaemon(conn, fmt.Sprintf("ADD_ONION NEW:BEST Port=%d,%s", port, DEFAULT_BIND_ADDRESS))
	response := readDaemonAnswer(conn)
	if len(response) > 0 {
		tokens := strings.Split(response[0], "=")
		if strings.Contains(tokens[0], "ServiceID") {
			ons.OnionAddress = fmt.Sprint(tokens[1], ".onion:", port)
			logging.LogInfo("Hidden Service started on", ons.OnionAddress)
		}
	}

	conn.Close()
}

func (ons *TorProto) Close() {
	conn := connectToControlPort(ons.TorControlPort, ons.TorControlPassword)
	logging.LogInfo("Removing Hidden Service", ons.OnionAddress)
	writeToDaemon(conn, fmt.Sprintf("DEL_ONION %s", strings.Split(ons.OnionAddress, ".")[0]))
	readDaemonAnswer(conn)
	conn.Close()
}

func (i2p *TorProto) Listen() (net.Listener, error) {
	port := randPort()
	return net.Listen("tcp", fmt.Sprintf("%s:%d", DEFAULT_BIND_ADDRESS, port))
}
