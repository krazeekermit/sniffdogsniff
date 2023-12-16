package i2p

import (
	"fmt"
	"net"
)

type I2pListener struct {
	ctx  I2PCtx
	id   string
	addr I2pAddr
	conn net.Conn
}

func (l I2pListener) Accept() (net.Conn, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", l.ctx.SamAPIPort))
	if err != nil {
		panic("failed to connect to the i2p daemon")
	}

	sendHelloCmd(l.ctx, conn)

	writeCommand(conn, SAM_CMD_STREAM, fmt.Sprintf(SAM_CMD_STREAM_ACCEPT_ARGS, l.id))
	reply, err := readSamReply(l.conn, SAM_CMD_SESSION)
	if err != nil {
		return nil, err
	}
	if len(reply) > 0 {
		return nil, fmt.Errorf("i2p: error accepting socket connection")
	}
	return conn, nil
}

func (l I2pListener) Addr() net.Addr {
	return l.addr
}

func (l I2pListener) Close() error {
	if l.conn != nil {
		return l.conn.Close()
	}
	return nil
}
