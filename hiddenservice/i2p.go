package hiddenservice

import (
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/sniffdogsniff/logging"
	"github.com/sniffdogsniff/util"
)

const I2P_PRIV_KEY_FILE_NAME = "i2pkey.dat"

const (
	SAM_CMD_HELLO               = "HELLO"
	SAM_CMD_HELLO_ARGS          = "VERSION MIN=3.1 MAX=3.1"
	SAM_CMD_HELLO_PASSWORD_ARGS = "VERSION MIN=3.1 MAX=3.1 USER=%s PASSWORD=%s"
	SAM_CMD_DEST                = "DEST"
	SAM_CMD_DEST_ARGS           = "GENERATE SIGNATURE_TYPE=7"
	SAM_CMD_SESSION             = "SESSION"
	SAM_CMD_SESSION_CREATE_ARGS = "CREATE STYLE=STREAM ID=%s DESTINATION=%s FROM_PORT=%d TO_PORT=%d"
	SAM_CMD_STREAM              = "STREAM"
	SAM_CMD_STREAM_ACCEPT_ARGS  = "ACCEPT ID=%s SILENT=false"
)

const (
	RESULT_ARG = "RESULT"
	PRIV_ARG   = "PRIV"
	PUB_ARG    = "PUB"
)

// Standard Base64 uses `+` and `/` as last two characters of its alphabet. * I2P Base64 uses `-` and `~` respectively.
func swapBase64(i2pB64 string) string {
	stdB64 := make([]byte, len(i2pB64))
	for i, c := range i2pB64 {
		switch c {
		case '-':
			stdB64[i] = '+'
		case '~':
			stdB64[i] = '/'
		case '+':
			stdB64[i] = '-'
		case '/':
			stdB64[i] = '~'
		default:
			stdB64[i] = i2pB64[i]
		}
	}
	return string(stdB64)
}

func toB32AddrStr(b64string string) string {
	b64bytes, err := base64.StdEncoding.DecodeString(swapBase64(b64string))
	if err != nil {
		logging.LogError("error decoding b64 address")
	}
	alen := binary.BigEndian.Uint16(b64bytes[385:387])
	b32bytes := make([]byte, 0)
	for _, e := range sha256.Sum256(b64bytes[:387+alen]) {
		b32bytes = append(b32bytes, e)
	}

	return fmt.Sprintf("%s.b32.i2p", strings.TrimRight(strings.ToLower(base32.StdEncoding.EncodeToString(b32bytes)), "="))
}

// remember to put all constants in const!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
func readSamReply(conn net.Conn, cmd string) (map[string]string, error) {
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
	replyStr := strings.TrimRight(string(recvBytes), "\n")

	toks := strings.Split(replyStr, " ")
	if toks[0] != cmd {
		return nil, fmt.Errorf("wrong response command")
	}

	kvMap := make(map[string]string)
	for _, tok := range toks[2:] {
		r_toks := strings.SplitN(tok, "=", 2)
		kvMap[r_toks[0]] = r_toks[1]
	}
	return kvMap, nil
}

type I2PProto struct {
	NeedAuth    bool
	SamAPIPort  int
	SamUser     string
	SamPassword string
	BindPort    int
	WorkDirPath string
	id          string
	b32addr     string
	conn        net.Conn
}

func (i2p *I2PProto) sendHelloCmd(conn net.Conn) {
	cmdArgs := SAM_CMD_HELLO_ARGS
	if i2p.SamUser != "" && i2p.SamPassword != "" {
		cmdArgs = fmt.Sprintf(SAM_CMD_HELLO_PASSWORD_ARGS, i2p.SamUser, i2p.SamPassword)
	}
	writeCommand(conn, SAM_CMD_HELLO, cmdArgs)
	helloReply, err := readSamReply(conn, SAM_CMD_HELLO)
	if err != nil {
		panic(err.Error())
	}
	if helloReply[RESULT_ARG] != TOR_REPLY_OK {
		panic("i2pd response not ok")
	}
}

func (i2p *I2PProto) Accept() (net.Conn, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", i2p.SamAPIPort))
	if err != nil {
		panic("failed to connect to the i2p daemon")
	}

	i2p.sendHelloCmd(conn)

	writeCommand(conn, SAM_CMD_STREAM, fmt.Sprintf(SAM_CMD_STREAM_ACCEPT_ARGS, i2p.id))
	reply, err := readSamReply(i2p.conn, SAM_CMD_SESSION)
	if err != nil {
		return nil, err
	}
	if len(reply) > 0 {
		return nil, fmt.Errorf("i2p: error accepting socket connection")
	}
	return conn, nil
}

func (i2p *I2PProto) Addr() net.Addr {
	return nil
}

// goSam is not working for our purposes so we need to implement it ourselves
// not fully implemented do not use it!
func (i2p *I2PProto) Listen() (net.Listener, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", i2p.SamAPIPort))
	if err != nil {
		panic("failed to connect to the i2p daemon")
	}

	i2p.sendHelloCmd(conn)

	dest := ""
	i2pKeyFilePath := filepath.Join(i2p.WorkDirPath, I2P_PRIV_KEY_FILE_NAME)

	if util.FileExists(i2pKeyFilePath) {
		fp, err := os.OpenFile(i2pKeyFilePath, os.O_RDONLY, 0600)
		if err != nil {
			fp.Close()
			panic("i2p failed to red keyblob file")
		}
		end, err := fp.Seek(0, os.SEEK_END)
		if err != nil {
			fp.Close()
			panic("i2p failed to red keyblob file")
		}
		_, err = fp.Seek(0, os.SEEK_SET)
		if err != nil {
			fp.Close()
			panic("i2p failed to red keyblob file")
		}
		d := make([]byte, end)
		fp.Read(d)
		dest = string(d)
		fp.Close()
	} else {
		writeCommand(conn, SAM_CMD_DEST, SAM_CMD_DEST_ARGS)
		destReply, err := readSamReply(conn, SAM_CMD_DEST)
		if err != nil {
			panic(err.Error())
		}

		d, present := destReply[PRIV_ARG]
		if !present {
			panic("no pub key i2p error")
		}
		dest = d

		fp, err := os.OpenFile(i2pKeyFilePath, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			fp.Close()
			panic(err.Error())
		}
		fp.WriteString(dest)
		fp.Close()
	}

	i2p.id = util.GenerateId3_Str()
	writeCommand(conn, SAM_CMD_SESSION, fmt.Sprintf(SAM_CMD_SESSION_CREATE_ARGS, i2p.id, dest, i2p.BindPort, i2p.BindPort))
	resp, err := readSamReply(conn, SAM_CMD_SESSION)
	if err != nil {
		panic(err.Error())
	}
	if resp[RESULT_ARG] != TOR_REPLY_OK {
		panic("error creating i2p session")
	}

	i2p.b32addr = toB32AddrStr(dest)
	i2p.conn = conn

	// From https://geti2p.net/spec/common-structures#destination

	// To implement a basic TCP-only, peer-to-peer application, the client must support the following commands.

	// HELLO VERSION MIN=3.1 MAX=3.1
	// reply : HELLO REPLY RESULT=OK VERSION=3.1
	// Needed for all of the remaining ones
	// DEST GENERATE SIGNATURE_TYPE=7
	// reply PUB==destination base64 needs to be converted to base32
	/**
			note: from bitcoin i2pd.cpp
			/**
	 * Swap Standard Base64 <-> I2P Base64.
	 * Standard Base64 uses `+` and `/` as last two characters of its alphabet.
	 * I2P Base64 uses `-` and `~` respectively.
	 * So it is easy to detect in which one is the input and convert to the other.
		**/

	// To generate our private key and destination
	// NAMING LOOKUP NAME=...
	// To convert .i2p addresses to destinations
	// SESSION CREATE STYLE=STREAM ID=... DESTINATION=...
	// alternative SESSION CREATE STYLE=STREAM ID=1226 DESTINATION=TRANSIENT SIGNATURE_TYPE=7
	// Needed for STREAM CONNECT and STREAM ACCEPT
	// STREAM CONNECT ID=... DESTINATION=...
	// To make outgoing connections
	// STREAM ACCEPT ID=...
	// To accept incoming connections

	return i2p, nil
}

func (i2p *I2PProto) GetAddressString() string {
	return fmt.Sprintf("%s:%d", i2p.b32addr, i2p.BindPort)
}

func (i2p *I2PProto) Close() error {
	return nil
}
