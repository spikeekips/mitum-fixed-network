package state

import (
	"encoding/json"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (sv SliceValue) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(struct {
		jsonenc.HintedHead
		H valuehash.Hash `json:"hash"`
		V []hint.Hinter  `json:"value"`
	}{
		HintedHead: jsonenc.NewHintedHead(sv.Hint()),
		H:          sv.Hash(),
		V:          sv.v,
	})
}

func (sv *SliceValue) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uv struct {
		H valuehash.Bytes   `json:"hash"`
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
