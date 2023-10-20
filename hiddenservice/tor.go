package hiddenservice

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sniffdogsniff/logging"
	"github.com/sniffdogsniff/util"
)

const TOR = "tor"

const KEY_BLOB_FILE_NAME = "onionkey.dat"

const (
	TOR_CODE_SUCCESS              = 250
	TOR_CODE_UNRECOGNIZED_COMMAND = 510
)

const (
	TOR_CMD_PROTOCOLINFO  = "PROTOCOLINFO"
	TOR_CMD_AUTHCHALLENGE = "AUTHCHALLENGE"
	TOR_CMD_AUTHENTICATE  = "AUTHENTICATE"
	TOR_CMD_ADD_ONION     = "ADD_ONION"
	TOR_CMD_DEL_ONION     = "DEL_ONION"

	TOR_NEW_KEYBLOB                   = "NEW"
	TOR_AUTH                          = "250-AUTH"
	TOR_FLAG_METHODS                  = "METHODS"
	TOR_METHOD_COOKIE                 = "COOKIE,SAFECOOKIE"
	TOR_HMAC_SERVER_TO_CONTROLLER_KEY = "Tor safe cookie authentication server-to-controller hash"
	TOR_HMAC_CONTROLLER_TO_SERVER_KEY = "Tor safe cookie authentication controller-to-server hash"
	TOR_ARG_SAFECOOKIE                = "SAFECOOKIE"
	TOR_FLAG_COOKIEFILE               = "COOKIEFILE"
	TOR_FLAG_SERVERHASH               = "SERVERHASH"
	TOR_FLAG_SERVERNONCE              = "SERVERNONCE"
	TOR_FLAG_PORT                     = "Port"
	TOR_ED25519_V3                    = "ED25519-V3"
	TOR_ONION_SERVICE_ID              = "250-ServiceID"
	TOR_ONION_SERVICE_PRIV_KEY        = "250-PrivateKey"
)

const (
	TOR_REPLY_OK string = "OK"
)

const TXT_BUFFER_SIZE = 2048
const DEFAULT_BIND_ADDRESS = "127.0.0.1"

func writeCommand(conn net.Conn, cmd, args string) error {
	cmdStr := fmt.Sprintf("%s %s", cmd, args)
	n, err := conn.Write(append([]byte(cmdStr), '\n'))
	if err != nil || n < len([]byte(cmdStr)) {
		return fmt.Errorf("can't communicate to daemon")
	}
	return nil
}

func readReply(conn net.Conn) ([]string, error) {
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
	respLines := make([]string, 0)
	for _, line := range strings.Split(string(recvBytes), "\n") {
		cleanedLine := strings.Trim(line, "\n\r")
		if len(cleanedLine) > 0 {
			respLines = append(respLines, cleanedLine)
		}
	}

	lastLineToks := strings.SplitN(respLines[len(respLines)-1], " ", 2)
	reply := lastLineToks[1]
	code, err := strconv.Atoi(lastLineToks[0])
	if err != nil {
		return make([]string, 0), err
	}

	if code == TOR_CODE_SUCCESS {
		if reply == TOR_REPLY_OK {
			return respLines[:len(respLines)-1], nil
		} else {
			return respLines, nil
		}
	} else {
		return make([]string, 0), fmt.Errorf("tor daemon error: %d %s", code, reply)
	}
}

func readReply_PROTOCOLINFO(conn net.Conn) (string, string, error) { // authmethod, cookiefile
	respLines, err := readReply(conn)
	if err != nil {
		return "", "", err
	}
	if len(respLines) < 3 {
		return "", "", fmt.Errorf("invalid response for PROTOCOLINFO")
	}

	method := ""
	cookieFilePath := ""

	toks := strings.Split(respLines[1], " ")
	if len(toks) == 3 && toks[0] == TOR_AUTH {
		ltoks := strings.Split(toks[1], "=")
		if ltoks[0] == TOR_FLAG_METHODS {
			method = ltoks[1]
		}
		ltoks = strings.Split(toks[2], "=")
		if ltoks[0] == TOR_FLAG_COOKIEFILE {
			cookieFilePath = strings.Trim(ltoks[1], "\"")
		}
	} else {
		return "", "", fmt.Errorf("unable to parse tor response")
	}

	return method, cookieFilePath, nil
}

func readReply_AUTHCHALLENGE(conn net.Conn) ([]byte, []byte, error) { // serverhash, servernonce
	respLines, err := readReply(conn)
	if err != nil {
		return []byte{}, []byte{}, err
	}
	if len(respLines) == 0 {
		return []byte{}, []byte{}, fmt.Errorf("invalid response for AUTHCHALLENGE")
	}

	var serverHash []byte
	var serverNonce []byte

	toks := strings.Split(respLines[0], " ")
	if len(toks) == 4 && toks[1] == TOR_CMD_AUTHCHALLENGE {
		ltoks := strings.Split(toks[2], "=")
		if ltoks[0] == TOR_FLAG_SERVERHASH {
			serverHash, err = hex.DecodeString(ltoks[1])
			if err != nil {
				return []byte{}, []byte{}, err
			}
		}
		ltoks = strings.Split(toks[3], "=")
		if ltoks[0] == TOR_FLAG_SERVERNONCE {
			serverNonce, err = hex.DecodeString(ltoks[1])
			if err != nil {
				return []byte{}, []byte{}, err
			}
		}
	} else {
		return []byte{}, []byte{}, fmt.Errorf("unable to parse tor response")
	}

	return serverHash, serverNonce, nil
}

func readReply_ADD_ONION(conn net.Conn) (string, string, error) {
	respLines, err := readReply(conn)
	if err != nil {
		return "", "", err
	}

	serviceId := ""
	privKey := ""

	if len(respLines) > 0 {
		toks := strings.SplitN(respLines[0], "=", 2)
		if toks[0] == TOR_ONION_SERVICE_ID {
			serviceId = toks[1]
		} else {
			return "", "", fmt.Errorf("unable to parse tor response")
		}
	}
	if len(respLines) > 1 {
		toks := strings.SplitN(respLines[1], "=", 2)
		if toks[0] == TOR_ONION_SERVICE_PRIV_KEY {
			privKey = toks[1]
		} else {
			return "", "", fmt.Errorf("unable to parse tor response")
		}
	}

	return serviceId, privKey, nil
}

type TorProto struct {
	TorControlPort     int
	TorControlPassword string
	TorCookieAuth      bool
	BindPort           int
	WorkDirPath        string
	onionId            string
	controlPortConn    net.Conn
}

func (ons *TorProto) connectAndCreateHiddenService() {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", ons.TorControlPort))
	if err != nil {
		panic("Failed to create hidden service can't connect to tor daemon")
	}
	password := fmt.Sprintf("\"%s\"", ons.TorControlPassword)
	if ons.TorCookieAuth {
		if writeCommand(conn, TOR_CMD_PROTOCOLINFO, "") != nil {
			panic("can't connect to tor daemon")
		}
		methods, cookieFilePath, err := readReply_PROTOCOLINFO(conn)
		if err != nil {
			panic(err.Error())
		}
		if methods != TOR_METHOD_COOKIE {
			panic(fmt.Sprintf("not using the cookie ath for tor using %s", methods))
		}

		cookieData := make([]byte, 32)
		fp, err := os.OpenFile(cookieFilePath, os.O_RDONLY, 0600)
		if err != nil {
			panic(err.Error())
		}
		n, err := fp.Read(cookieData)
		if err != nil {
			fp.Close()
			panic(err.Error())
		}
		if n != 32 {
			fp.Close()
			panic("tor cookie file must be 32bytes long")
		}
		fp.Close()

		clientNonce := make([]byte, 32)
		rand.Read(clientNonce)

		if writeCommand(conn, TOR_CMD_AUTHCHALLENGE, fmt.Sprintf("%s %s", TOR_ARG_SAFECOOKIE, hex.EncodeToString(clientNonce))) != nil {
			panic("can't connect to tor daemon")
		}
		serverHash, serverNonce, err := readReply_AUTHCHALLENGE(conn)
		if err != nil {
			panic(err.Error())
		}

		srv2ctrlHmac := hmac.New(sha256.New, []byte(TOR_HMAC_SERVER_TO_CONTROLLER_KEY))
		srv2ctrlHmac.Write(append(append(cookieData, clientNonce...), serverNonce...))
		eServerHash := srv2ctrlHmac.Sum(nil)
		if !bytes.Equal(serverHash, eServerHash) {
			panic("tor auth: wrong server hash")
		}

		ctrl2srvHmac := hmac.New(sha256.New, []byte(TOR_HMAC_CONTROLLER_TO_SERVER_KEY))
		ctrl2srvHmac.Write(append(append(cookieData, clientNonce...), serverNonce...))
		password = hex.EncodeToString(ctrl2srvHmac.Sum(nil))
	}
	if writeCommand(conn, TOR_CMD_AUTHENTICATE, password) != nil {
		panic("can't connect to tor daemon")
	}
	_, err = readReply(conn)
	if err != nil {
		panic(err.Error())
	}

	keyBlobFilePath := filepath.Join(ons.WorkDirPath, KEY_BLOB_FILE_NAME)

	keyArgs := fmt.Sprintf("%s:%s", TOR_NEW_KEYBLOB, TOR_ED25519_V3)
	cachedKeyBlob := util.FileExists(keyBlobFilePath)
	if cachedKeyBlob {
		fp, err := os.OpenFile(keyBlobFilePath, os.O_RDONLY, 0600)
		if err != nil {
			logging.Errorf(TOR, "Failed to open Tor keyblob file: %s", err.Error())
		} else {
			buf := make([]byte, 256)
			n, err := fp.Read(buf)
			if err != nil {
				logging.Errorf(TOR, "Failed to read Tor keyblob file")
			} else {
				base64Key := base64.StdEncoding.EncodeToString(buf[:n])
				keyArgs = fmt.Sprintf("%s:%s", TOR_ED25519_V3, base64Key)
				logging.Debugf(TOR, "using cached Tor keyblob file key %s", keyArgs)
			}
			fp.Close()
		}
	}
	if writeCommand(conn, TOR_CMD_ADD_ONION, fmt.Sprintf("%s %s=%d,%s:%d", keyArgs, TOR_FLAG_PORT,
		ons.BindPort, DEFAULT_BIND_ADDRESS, ons.BindPort)) != nil {
		panic("can't connect to tor daemon")
	}

	serviceId, privKey, err := readReply_ADD_ONION(conn)
	if err != nil {
		panic(err.Error())
	}

	// successfully created onion service
	ons.controlPortConn = conn

	if !cachedKeyBlob {
		fp, err := os.OpenFile(keyBlobFilePath, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			logging.Errorf(TOR, "Failed to write Tor keyblob file: %s", err.Error())
		} else {
			key := strings.Split(privKey, ":")[1]
			keyBytes, err := base64.StdEncoding.DecodeString(key)
			if err != nil {
				logging.Errorf(TOR, "Tor: cannod decode private key base64 string")
			} else {
				fp.Write(keyBytes)
			}
		}
		fp.Close()
	}

	ons.onionId = serviceId
	logging.Infof(TOR, "Created onion service at %s", ons.onionId)
}

func (ons *TorProto) Close() error {
	logging.Infof(TOR, "Removing onion service %s", ons.onionId)
	if writeCommand(ons.controlPortConn, TOR_CMD_DEL_ONION, ons.onionId) != nil {
		logging.Errorf(TOR, "Failed removing onion service")
	}
	_, err := readReply(ons.controlPortConn)
	if err != nil {
		logging.Errorf(TOR, "Failed removing onion service:", err.Error())
	}
	ons.controlPortConn.Close()
	return nil
}

func (ons *TorProto) Listen() (net.Listener, error) {
	ons.connectAndCreateHiddenService()
	return net.Listen("tcp", fmt.Sprintf("%s:%d", DEFAULT_BIND_ADDRESS, ons.BindPort))
}

func (ons *TorProto) GetAddressString() string {
	return fmt.Sprintf("%s.onion:%d", ons.onionId, ons.BindPort)
}
