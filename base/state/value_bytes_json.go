package state

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (bv BytesValue) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(struct {
		jsonenc.HintedHead
		H valuehash.Hash `json:"hash"`
		V []byte         `json:"value"`
	}{
		HintedHead: jsonenc.NewHintedHead(bv.Hint()),
		H:          bv.h,
		V:          bv.v,
	})
}

func (bv *BytesValue) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uv struct {
		H valuehash.Bytes `json:"hash"`
		V []byte          `json:"value"`
	}

	if err := enc.Unmarshal(b, &uv); err != nil {
		return err
	}

	bv.h = uv.H
	bv.v = uv.V

	return nil
}
