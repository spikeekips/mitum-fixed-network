package state

import (
	"encoding/json"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

type StateV0AVLNodePackerJSON struct {
	encoder.JSONPackHintedHead
	H   []byte   `json:"hash"`
	K   []byte   `json:"key"`
	HT  int16    `json:"height"`
	LF  []byte   `json:"left_key"`
	LFH []byte   `json:"left_hash"`
	RG  []byte   `json:"right_key"`
	RGH []byte   `json:"right_hash"`
	ST  *StateV0 `json:"state"`
}

func (stav StateV0AVLNode) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(StateV0AVLNodePackerJSON{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(stav.Hint()),
		H:                  stav.h,
		K:                  stav.Key(),
		HT:                 stav.height,
		LF:                 stav.left,
		LFH:                stav.leftHash,
		RG:                 stav.right,
		RGH:                stav.rightHash,
		ST:                 stav.state,
	})
}

type StateV0AVLNodeUnpackerJSON struct {
	H   []byte          `json:"hash"`
	HT  int16           `json:"height"`
	LF  []byte          `json:"left_key"`
	LFH []byte          `json:"left_hash"`
	RG  []byte          `json:"right_key"`
	RGH []byte          `json:"right_hash"`
	ST  json.RawMessage `json:"state"`
}

func (stav *StateV0AVLNode) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var us StateV0AVLNodeUnpackerJSON
	if err := enc.Unmarshal(b, &us); err != nil {
		return err
	}

	var state StateV0
	if s, err := DecodeState(enc, us.ST); err != nil {
		return err
	} else if sv, ok := s.(StateV0); !ok {
		return hint.InvalidTypeError.Errorf("not state.StateV0; type=%T", s)
	} else {
		state = sv
	}

	stav.h = us.H
	stav.height = us.HT
	stav.h = us.H
	stav.left = us.LF
	stav.leftHash = us.LFH
	stav.right = us.RG
	stav.rightHash = us.RGH
	stav.state = &state

	return nil
}
