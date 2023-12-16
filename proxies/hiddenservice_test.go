package proxies_test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/sniffdogsniff/logging"
	"github.com/sniffdogsniff/proxies/i2p"
	"github.com/sniffdogsniff/proxies/tor"
	"github.com/sniffdogsniff/util"
	"golang.org/x/net/proxy"
)

const TOR_PASSWORD = "test1234"

/*
	Tor daemon setup for testing
	Edit your /etc/tor/torrc or /usr/local/etc/tor/torrc
	and add the following lines:
	ControlPort 9051
	HashedControlPassword 16:0B3747351005A87D6041C2B6DD8E96ABAB116A15395E0E3070E8300828

	!!! DO NOT LEAVE YOUR TOR PASSWORD AS THE TEST PASSWORD !!!

	This test requires tor daemon running
	to start tor: service tor start | onestart
*/

func Test_Onion_NewKeyBlob(t *testing.T) {
	logging.InitLogging(logging.DEBUG)

	torControl := tor.NewTorControlSession()
	onionAddr, err := torControl.CreateOnionService(tor.TorCtx{
		TorControlPort:     9051,
		TorControlPassword: TOR_PASSWORD,
		WorkDirPath:        "./",
		BindPort:           5009,
	}, 1234, "")

	if err != nil {
		t.Fatalf(err.Error())
	}

	l, err := net.Listen("tcp", "127.0.0.1:1234")
	if err != nil {
		t.Fatalf(err.Error())
	}

	go func(l net.Listener) {

		conn, err := l.Accept()
		if err != nil {
			panic("test failed")
		}
		conn.Write([]byte("hello1234"))

	}(l)

	dialer, err := proxy.SOCKS5("tcp", "127.0.0.1:9050", nil, &net.Dialer{
		Timeout:   60 * time.Second,
		KeepAlive: 30 * time.Second,
	})
	if err != nil {
		t.Fatalf(err.Error())
	}
	fmt.Println("a", onionAddr)
	conn, err := dialer.Dial("tcp", onionAddr)
	if err != nil {
		t.Fatalf(err.Error())
	}

	bytez := make([]byte, 120)
	n, err := conn.Read(bytez)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if n == 0 {
		t.Fatal()
	}

	if string(bytez[:n]) != "hello1234" {
		t.Fatal()
	}

	if !util.FileExists(tor.KEY_BLOB_FILE_NAME) {
		t.Fatal()
	}

	torControl.DeleteOnion()
}

func Test_I2P(t *testing.T) {
	logging.InitLogging(logging.DEBUG)
	ctx := i2p.I2PCtx{
		SamAPIPort:  7656,
		SamUser:     "",
		SamPassword: "",
	}

	samSession, err := i2p.NewI2PSamSession(ctx, "", 1234)
	if err != nil {
		t.Fatalf(err.Error())
	}

	l, err := net.Listen("tcp", "127.0.0.1:1234")
	if err != nil {
		t.Fatalf(err.Error())
	}

	go func(l net.Listener) {

		conn, err := l.Accept()
		fmt.Println("Accept!!!!!!!!!!!!!")
		if err != nil {
			panic("test failed")
		}
		conn.Write([]byte("hello1234"))

	}(l)

	fmt.Println("a", samSession.Base32Addr)
	clientSam, err := i2p.NewI2PSamSession_Transient(ctx)
	if err != nil {
		t.Fatalf(err.Error())
	}

	conn, err := clientSam.I2PDial(samSession.Base32Addr)
	if err != nil {
		t.Fatalf(err.Error())
	}

	bytez := make([]byte, 120)
	n, err := conn.Read(bytez)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if n == 0 {
		t.Fatal()
	}

	if string(bytez[:n]) != "hello1234" {
		t.Fatal()
	}
}
