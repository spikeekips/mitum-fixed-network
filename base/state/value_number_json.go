package state

import (
	"encoding/json"
	"reflect"

	"github.com/spikeekips/mitum/base/valuehash"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type NumberValueJSONPacker struct {
	jsonenc.HintedHead
	H valuehash.Hash `json:"hash"`
	V []byte         `json:"value"`
	T reflect.Kind   `json:"type"`
}

func (nv NumberValue) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(NumberValueJSONPacker{
		HintedHead: jsonenc.NewHintedHead(nv.Hint()),
		H:          nv.Hash(),
		V:          nv.b,
		T:          nv.t,
	})
}

type NumberValueJSONUnpacker struct {
	H json.RawMessage `json:"hash"`
	V []byte          `json:"value"`
	T reflect.Kind    `json:"type"`
}

func (nv *NumberValue) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uv NumberValueJSONUnpacker
	if err := enc.Unmarshal(b, &uv); err != nil {
		return err
	}

	return nv.unpack(enc, uv.H, uv.V, uv.T)
}
