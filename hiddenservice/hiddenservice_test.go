package hiddenservice_test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/sniffdogsniff/hiddenservice"
	"github.com/sniffdogsniff/logging"
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
	logging.InitLogging(logging.TRACE)

	torProto := hiddenservice.TorProto{
		TorControlPort:     9051,
		TorControlPassword: TOR_PASSWORD,
		WorkDirPath:        "./",
		BindPort:           5009,
	}

	l, err := torProto.Listen()
	if err != nil {
		panic("test failed")
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
	fmt.Println("a", torProto.GetAddressString())
	conn, err := dialer.Dial("tcp", torProto.GetAddressString())
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

	if !util.FileExists(hiddenservice.KEY_BLOB_FILE_NAME) {
		t.Fatal()
	}

	torProto.Close()
}
