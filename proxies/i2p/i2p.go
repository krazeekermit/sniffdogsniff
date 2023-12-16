package i2p

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

const I2P = "i2p"

const I2P_PRIV_KEY_FILE_NAME = "i2pkey.dat"

const (
	SAM_CMD_HELLO               = "HELLO"
	SAM_CMD_HELLO_ARGS          = "VERSION MIN=3.1 MAX=3.1"
	SAM_CMD_HELLO_PASSWORD_ARGS = "VERSION MIN=3.1 MAX=3.1 USER=%s PASSWORD=%s"
	SAM_CMD_NAMING              = "NAMING"
	SAM_CMD_NAMING_ARGS         = "LOOKUP NAME=%s"
	SAM_CMD_DEST                = "DEST"
	SAM_CMD_DEST_ARGS           = "GENERATE SIGNATURE_TYPE=7"
	SAM_CMD_SESSION             = "SESSION"
	SAM_CMD_SESSION_CREATE_ARGS = "CREATE STYLE=STREAM ID=%s DESTINATION=%s"
	SAM_CMD_FROM_TO_PORT_ARGS   = "FROM_PORT=%d TO_PORT=%d"
	SAM_CMD_TRANSIENT_ARG       = "TRANSIENT"
	SAM_CMD_STREAM              = "STREAM"
	SAM_CMD_STREAM_ACCEPT_ARGS  = "ACCEPT ID=%s SILENT=false"
	SAM_CMD_STREAM_CONNECT_ARGS = "CONNECT ID=%s DESTINATION=%s"
)

const (
	RESULT_ARG = "RESULT"
	VALUE_ARG  = "VALUE"
	PRIV_ARG   = "PRIV"
	PUB_ARG    = "PUB"
)

const (
	I2P_REPLY_OK string = "OK"
)

const TXT_BUFFER_SIZE = 2048
const DEFAULT_BIND_ADDRESS = "127.0.0.1"

func writeCommand(conn net.Conn, cmd, args string) error {
	cmdStr := fmt.Sprintf("%s %s", cmd, args)
	fmt.Printf("debug cmd:: %s\n", cmdStr)
	n, err := conn.Write(append([]byte(cmdStr), '\n'))
	if err != nil || n < len([]byte(cmdStr)) {
		return fmt.Errorf("can't communicate to daemon")
	}
	return nil
}

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
		logging.Errorf(I2P, "error decoding b64 address")
	}
	alen := binary.BigEndian.Uint16(b64bytes[385:387])
	b32bytes := make([]byte, 0)
	for _, e := range sha256.Sum256(b64bytes[:387+alen]) {
		b32bytes = append(b32bytes, e)
	}

	return fmt.Sprintf("%s.b32.i2p", strings.TrimRight(strings.ToLower(base32.StdEncoding.EncodeToString(b32bytes)), "="))
}

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
	fmt.Println("[SAM REPLY]", replyStr)

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

type I2PCtx struct {
	NeedAuth    bool
	SamAPIPort  int
	SamUser     string
	SamPassword string
	BindPort    int
	WorkDirPath string
}

func sendHelloCmd(i2p I2PCtx, conn net.Conn) error {
	cmdArgs := SAM_CMD_HELLO_ARGS
	if i2p.SamUser != "" && i2p.SamPassword != "" {
		cmdArgs = fmt.Sprintf(SAM_CMD_HELLO_PASSWORD_ARGS, i2p.SamUser, i2p.SamPassword)
	}
	writeCommand(conn, SAM_CMD_HELLO, cmdArgs)
	helloReply, err := readSamReply(conn, SAM_CMD_HELLO)
	if err != nil {
		return err
	}
	if helloReply[RESULT_ARG] != I2P_REPLY_OK {
		return fmt.Errorf("i2pd response not ok")
	}
	return nil
}

type I2pAddr struct {
	b32  string
	port int
}

func (a I2pAddr) String() string {
	return fmt.Sprintf("%s:%d", a.b32, a.port)
}

func (a I2pAddr) Network() string {
	return "tcp"
}

type I2PSamSession struct {
	ctx        I2PCtx
	id         string
	Base32Addr string
	conn       net.Conn
}

func NewI2PSamSession_Transient(ctx I2PCtx) (*I2PSamSession, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", ctx.SamAPIPort))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the i2p daemon")
	}

	sendHelloCmd(ctx, conn)

	s := &I2PSamSession{
		ctx:  ctx,
		id:   util.GenerateId3_Str(),
		conn: conn,
	}
	writeCommand(conn, SAM_CMD_SESSION, fmt.Sprintf(SAM_CMD_SESSION_CREATE_ARGS, s.id, SAM_CMD_TRANSIENT_ARG))
	resp, err := readSamReply(conn, SAM_CMD_SESSION)
	if err != nil {
		return nil, err
	}
	if resp[RESULT_ARG] != I2P_REPLY_OK {
		return nil, fmt.Errorf("error creating i2p session")
	}

	return s, nil
}

func NewI2PSamSession(ctx I2PCtx, workDir string, port int) (*I2PSamSession, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", ctx.SamAPIPort))
	if err != nil {
		panic("failed to connect to the i2p daemon")
	}

	sendHelloCmd(ctx, conn)

	dest := ""
	i2pKeyFilePath := filepath.Join(ctx.WorkDirPath, I2P_PRIV_KEY_FILE_NAME)

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

	s := &I2PSamSession{
		ctx:        ctx,
		Base32Addr: toB32AddrStr(dest),
		id:         util.GenerateId3_Str(),
		conn:       conn,
	}

	writeCommand(conn, SAM_CMD_SESSION, fmt.Sprint(
		fmt.Sprintf(SAM_CMD_SESSION_CREATE_ARGS, s.id, dest), // " ",
		//fmt.Sprintf(SAM_CMD_FROM_TO_PORT_ARGS, port, port),
	))
	resp, err := readSamReply(conn, SAM_CMD_SESSION)
	if err != nil {
		panic(err.Error())
	}
	fmt.Println(resp)
	if resp[RESULT_ARG] != I2P_REPLY_OK {
		panic("error creating i2p session")
	}

	return s, nil
}

func (s *I2PSamSession) I2PDial(b32addr string) (net.Conn, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", s.ctx.SamAPIPort))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the i2p daemon")
	}

	sendHelloCmd(s.ctx, conn)

	// host, _, err := net.SplitHostPort(b32addr)
	// if err != nil {
	// 	return nil, err
	// }

	// port, err := strconv.Atoi(portStr)
	// if err != nil {
	// 	return nil, err
	// }

	writeCommand(conn, SAM_CMD_NAMING, fmt.Sprintf(SAM_CMD_NAMING_ARGS, b32addr))
	resp, err := readSamReply(conn, SAM_CMD_NAMING)
	if err != nil {
		return nil, err
	}
	if resp[RESULT_ARG] != I2P_REPLY_OK {
		return nil, fmt.Errorf("error i2p naming lookup")
	}
	destRaw, ok := resp[VALUE_ARG]
	if !ok {
		return nil, fmt.Errorf("error i2p naming lookup")
	}

	writeCommand(conn, SAM_CMD_STREAM, fmt.Sprint(
		fmt.Sprintf(SAM_CMD_STREAM_CONNECT_ARGS, s.id, destRaw), // " ",
		//fmt.Sprintf(SAM_CMD_FROM_TO_PORT_ARGS, port, port),
	))
	reply, err := readSamReply(conn, SAM_CMD_STREAM)
	if err != nil {
		return nil, err
	}
	_ = reply

	return conn, nil
}

func (s *I2PSamSession) Listener(port int) I2pListener {
	return I2pListener{
		ctx: s.ctx,
		id:  s.id,
		addr: I2pAddr{
			b32:  s.Base32Addr,
			port: port,
		},
	}
}
