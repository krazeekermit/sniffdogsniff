package util

import (
	"bytes"
	"compress/zlib"
	"io/ioutil"
	"os"
)

func SliceContains[T comparable](s []T, e T) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}
	return false
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

func DirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}
