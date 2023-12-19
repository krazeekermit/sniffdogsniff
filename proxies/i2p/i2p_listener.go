package i2p

import (
	"fmt"
	"net"
)

type I2pListener struct {
	ctx  I2PCtx
	id   string
	addr I2pAddr
}

func (l I2pListener) Accept() (net.Conn, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", l.ctx.SamAPIPort))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the i2p daemon")
	}

	sendHelloCmd(l.ctx, conn)

	writeCommand(conn, SAM_CMD_STREAM, fmt.Sprintf(SAM_CMD_STREAM_ACCEPT_ARGS, l.id))
	_, err = readSamReply(conn, SAM_CMD_STREAM)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, TXT_BUFFER_SIZE)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return nil, err
		}
		if n < TXT_BUFFER_SIZE {
			break
		}
	}
	return conn, nil
}

func (l I2pListener) Addr() net.Addr {
	return l.addr
}

func (l I2pListener) Close() error {
	return nil
}
