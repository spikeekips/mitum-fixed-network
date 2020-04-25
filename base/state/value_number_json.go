package state

import (
	"encoding/json"
	"reflect"

	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

type NumberValueJSONPacker struct {
	encoder.JSONPackHintedHead
	H valuehash.Hash `json:"hash"`
	V []byte         `json:"value"`
	T reflect.Kind   `json:"type"`
}

func (nv NumberValue) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(NumberValueJSONPacker{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(nv.Hint()),
		H:                  nv.Hash(),
		V:                  nv.b,
		T:                  nv.t,
	})
}

type NumberValueJSONUnpacker struct {
	H json.RawMessage `json:"hash"`
	V []byte          `json:"value"`
	T reflect.Kind    `json:"type"`
}

func (nv *NumberValue) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var uv NumberValueJSONUnpacker
	if err := enc.Unmarshal(b, &uv); err != nil {
		return err
	}

	return nv.unpack(enc, uv.H, uv.V, uv.T)
}
