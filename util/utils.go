package util

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"io/ioutil"
	"os"
)

func Array32Contains(array [][32]byte, entry [32]byte) bool {
	for _, elem := range array {
		if bytes.Equal(elem[:], entry[:]) {
			return true
		}
	}
	return false
}

func SliceContains[T comparable](s []T, e T) bool {
	for _, v := range s {
		if v == e {
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

func ZlibDecompress(data []byte) ([]byte, error) {
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

func ZlibCompress(data []byte) ([]byte, error) {
	var b bytes.Buffer

	w := zlib.NewWriter(&b)
	w.Write(data)
	w.Close()
	return b.Bytes(), nil
}

func Array32ToSlice(data [32]byte) []byte {
	slice := make([]byte, 0)
	for _, b := range data {
		slice = append(slice, b)
	}
	return slice
}

func HashToB64UrlsafeString(hash [32]byte) string {
	return base64.URLEncoding.EncodeToString(hash[:])
}

func B64UrlsafeStringToHash(b64 string) [32]byte {
	bytes, _ := base64.URLEncoding.DecodeString(b64)
	return SliceToArray32(bytes)
}

func MergeMaps[K comparable, V interface{}](m1, m2 map[K]V) map[K]V {
	for k, v := range m2 {
		m1[k] = v
	}
	return m1
}

func TwoUint64ToArr(a, b uint64) [2]uint64 {
	return [2]uint64{a, b}
}

func ArrToTwoUint64(arr [2]uint64) (uint64, uint64) {
	return arr[0], arr[1]
}

func MapToSlice[K comparable, V interface{}](m map[K]V) []V {
	s := make([]V, 0)
	for _, v := range m {
		s = append(s, v)
	}
	return s
}

func MapKeys[K comparable, V interface{}](m map[K]V) []K {
	s := make([]K, 0)
	for k := range m {
		s = append(s, k)
	}
	return s
}

/*
	File Utilities
*/

func FileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
