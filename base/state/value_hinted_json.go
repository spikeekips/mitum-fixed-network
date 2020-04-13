package state

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

func (hv HintedValue) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(struct {
		encoder.JSONPackHintedHead
		H valuehash.Hash `json:"hash"`
		V hint.Hinter    `json:"value"`
	}{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(hv.Hint()),
		H:                  hv.Hash(),
		V:                  hv.v,
	})
}

func (hv *HintedValue) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
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
