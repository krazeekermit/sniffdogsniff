package sds

import (
	"bytes"
	"compress/zlib"
	"io/ioutil"
)

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
