package core

import (
	"bytes"
	"encoding/gob"
)

func InitializeGob() {
	gob.Register(Peer{})
	gob.Register([]Peer{})
	gob.Register(SearchResult{})
	gob.Register([]SearchResult{})
	gob.Register(ResultMeta{})
	gob.Register([]ResultMeta{})
}

func GobMarshal(v interface{}) ([]byte, error) {
	b := new(bytes.Buffer)
	enc := gob.NewEncoder(b)
	err := enc.Encode(v)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func GobUnmarshal(data []byte, v interface{}) error {
	b := bytes.NewBuffer(data)
	return gob.NewDecoder(b).Decode(v)
}
