package state

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

func (sv SliceValue) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(struct {
		encoder.JSONPackHintedHead
		H valuehash.Hash `json:"hash"`
		V []hint.Hinter  `json:"value"`
	}{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(sv.Hint()),
		H:                  sv.Hash(),
		V:                  sv.v,
	})
}

func (sv *SliceValue) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var uv struct {
		H json.RawMessage   `json:"hash"`
		V []json.RawMessage `json:"value"`
	}

	if err := enc.Unmarshal(b, &uv); err != nil {
		return err
	}

	bValue := make([][]byte, len(uv.V))
	for i, v := range uv.V {
		bValue[i] = v
	}

	return sv.unpack(enc, uv.H, bValue)
}
