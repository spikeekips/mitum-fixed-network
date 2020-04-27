package state

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base/valuehash"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
)

func (bv BytesValue) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(struct {
		jsonencoder.HintedHead
		H valuehash.Hash `json:"hash"`
		V []byte         `json:"value"`
	}{
		HintedHead: jsonencoder.NewHintedHead(bv.Hint()),
		H:          bv.h,
		V:          bv.v,
	})
}

func (bv *BytesValue) UnpackJSON(b []byte, enc *jsonencoder.Encoder) error {
	var uv struct {
		H json.RawMessage `json:"hash"`
		V []byte          `json:"value"`
	}

	if err := enc.Unmarshal(b, &uv); err != nil {
		return err
	}

	var err error
	var h valuehash.Hash
	if h, err = valuehash.Decode(enc, uv.H); err != nil {
		return err
	}

	bv.h = h
	bv.v = uv.V

	return nil
}
