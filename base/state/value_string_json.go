package state

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base/valuehash"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

func (sv StringValue) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(struct {
		jsonenc.HintedHead
		H valuehash.Hash `json:"hash"`
		V string         `json:"value"`
	}{
		HintedHead: jsonenc.NewHintedHead(sv.Hint()),
		H:          sv.Hash(),
		V:          sv.v,
	})
}

func (sv *StringValue) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uv struct {
		H json.RawMessage `json:"hash"`
		V string          `json:"value"`
	}

	if err := enc.Unmarshal(b, &uv); err != nil {
		return err
	}

	var err error
	var h valuehash.Hash
	if h, err = valuehash.Decode(enc, uv.H); err != nil {
		return err
	}

	sv.h = h
	sv.v = uv.V

	return nil
}
