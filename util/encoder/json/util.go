package jsonenc

import (
	"encoding/json"

	"github.com/spikeekips/mitum/util"
)

func Marshal(v interface{}) ([]byte, error) {
	return util.JSON.Marshal(v)
}

func Unmarshal(b []byte, v interface{}) error {
	return util.JSON.Unmarshal(b, v)
}

func MustMarshal(i interface{}) []byte {
	b, _ := Marshal(i)

	return b
}

func MarshalIndent(i interface{}) ([]byte, error) {
	return json.MarshalIndent(i, "", "  ")
}

func MustMarshalIndent(i interface{}) []byte {
	b, _ := MarshalIndent(i)

	return b
}

func ToString(i interface{}) string {
	return string(MustMarshal(i))
}
