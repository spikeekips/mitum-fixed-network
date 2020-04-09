package state

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

func (bv BytesValue) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(struct {
		encoder.JSONPackHintedHead
		H valuehash.Hash `json:"hash"`
		V []byte         `json:"value"`
	}{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(bv.Hint()),
		H:                  bv.h,
		V:                  bv.v,
	})
}

func (bv *BytesValue) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
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
