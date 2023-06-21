package hiddenservice

import (
	"fmt"
	"net"
	"strings"

	"github.com/sniffdogsniff/util/logging"
)

type OnionService struct {
	NeedAuth           bool
	TorControlPort     int
	TorControlPassword string
	OnionAddress       string
	// SamPort            int
	// SamUser            string
	// SamPassword        string
}

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
	logging.LogTrace("Send command to daemon:", text)
	n, err := conn.Write([]byte(text + "\n"))
	if err != nil || n < len([]byte(text)) {
		panic("Failed to create hidden service can't communicate to daemon")
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

func (ons *OnionService) CreateHiddenService(bindAddress string) {
	conn := connectToControlPort(ons.TorControlPort,
		ons.TorControlPassword)

	addr, err := net.ResolveTCPAddr("tcp", bindAddress)
	if err != nil {
		panic(fmt.Sprintf("Malformed url: %s", bindAddress))
	}

	writeToDaemon(conn, fmt.Sprintf("ADD_ONION NEW:BEST Port=%d,%s", addr.Port, bindAddress))
	response := readDaemonAnswer(conn)
	if len(response) > 0 {
		tokens := strings.Split(response[0], "=")
		if strings.Contains(tokens[0], "ServiceID") {
			ons.OnionAddress = fmt.Sprint(tokens[1], ".onion:", addr.Port)
			logging.LogInfo("Hidden Service started on", ons.OnionAddress)
		}
	}

	conn.Close()
}

func (ons *OnionService) RemoveHiddenService() {
	conn := connectToControlPort(ons.TorControlPort, ons.TorControlPassword)
	logging.LogInfo("Removing Hidden Service", ons.OnionAddress)
	writeToDaemon(conn, fmt.Sprintf("DEL_ONION %s", strings.Split(ons.OnionAddress, ".")[0]))
	readDaemonAnswer(conn)
	conn.Close()
}
