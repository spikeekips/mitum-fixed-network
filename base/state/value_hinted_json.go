package state

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base/valuehash"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
)

func (hv HintedValue) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(struct {
		jsonenc.HintedHead
		H valuehash.Hash `json:"hash"`
		V hint.Hinter    `json:"value"`
	}{
		HintedHead: jsonenc.NewHintedHead(hv.Hint()),
		H:          hv.Hash(),
		V:          hv.v,
	})
}

func (hv *HintedValue) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uv struct {
		V json.RawMessage `json:"value"`
	}

	if err := enc.Unmarshal(b, &uv); err != nil {
		return err
	}

	decoded, err := enc.DecodeByHint(uv.V)
	if err != nil {
		return err
	}

	hv.v = decoded

	return nil
}
