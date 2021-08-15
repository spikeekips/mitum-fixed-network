package state

import (
	"github.com/pkg/errors"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
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
		H valuehash.Bytes `json:"hash"`
		V string          `json:"value"`
	}

	if err := enc.Unmarshal(b, &uv); err != nil {
		return err
	}

	if uv.H.IsEmpty() {
		return errors.Errorf("empty previous_block hash found")
	}

	sv.h = uv.H
	sv.v = uv.V

	return nil
}
