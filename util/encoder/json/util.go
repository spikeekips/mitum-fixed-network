package jsonenc

import (
	"bytes"
	"encoding/json"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
)

func Marshal(i interface{}) ([]byte, error) {
	return util.JSON.Marshal(i)
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

func Unmarshal(b []byte, i interface{}) error {
	return util.JSON.Unmarshal(b, i)
}

type HintedHead struct {
	H hint.Hint `json:"_hint"`
}

func NewHintedHead(h hint.Hint) HintedHead {
	return HintedHead{H: h}
}

func isNullRawMesage(b []byte) bool {
	if len(b) != jsonNULLLength {
		return false
	}

	return bytes.Equal(jsonNULL, b)
}
