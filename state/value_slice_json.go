package state

import (
	"encoding/json"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
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

	if i, err := valuehash.Decode(enc, uv.H); err != nil {
		return err
	} else {
		sv.h = i
	}

	v := make([]hint.Hinter, len(uv.V))
	for i, r := range uv.V {
		decoded, err := enc.DecodeByHint(r)
		if err != nil {
			return err
		}

		v[i] = decoded
	}

	if usv, err := (SliceValue{}).set(v); err != nil {
		return err
	} else {
		sv.b = usv.b
	}

	sv.v = v

	return nil
}
