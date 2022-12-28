package sds

import (
	"bytes"
	"compress/zlib"
	"io/ioutil"
	"log"
)

const (
	ANSI_RED    string = "\033[31m"
	ANSI_YELLOW string = "\033[33m"
	ANSI_WHITE  string = "\033[97m"
	ANSI_CYAN   string = "\033[36m"
	ANSI_END    string = "\033[0m"
)

func logInfo(text string) {
	log.Println(ANSI_WHITE, "[INFO]", text, ANSI_END)
}

func LogInfo(text string) {
	log.Println(ANSI_WHITE, "[INFO]", text, ANSI_END)
}

func logWarn(text string) {
	log.Println(ANSI_YELLOW, "[WARN]", text, ANSI_END)
}

func logError(text string) {
	log.Println(ANSI_RED, "[ERROR]", text, ANSI_END)
}

func Array32Contains(array [][32]byte, entry [32]byte) bool {
	for _, elem := range array {
		if bytes.Equal(elem[:], entry[:]) {
			return true
		}
	}
	return false
}

func SliceToArray32(data []byte) [32]byte {
	var bytes [32]byte
	for i, b := range data {
		if i > 31 {
			break
		}
		bytes[i] = b
	}
	return bytes
}

func zlibDecompress(data []byte) ([]byte, error) {
	b := bytes.NewReader(data)
	z, err := zlib.NewReader(b)
	if err != nil {
		return nil, err
	}
	defer z.Close()
	p, err := ioutil.ReadAll(z)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func zlibCompress(data []byte) ([]byte, error) {
	var b bytes.Buffer

	w := zlib.NewWriter(&b)
	w.Write(data)
	w.Close()
	return b.Bytes(), nil
}

func array32ToSlice(data [32]byte) []byte {
	slice := make([]byte, 0)
	for _, b := range data {
		slice = append(slice, b)
	}
	return slice
}
