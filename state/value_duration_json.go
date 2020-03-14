package state

import (
	"encoding/json"
	"time"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

func (dv DurationValue) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(struct {
		encoder.JSONPackHintedHead
		H valuehash.Hash `json:"hash"`
		V int64          `json:"value"`
	}{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(dv.Hint()),
		H:                  dv.Hash(),
		V:                  dv.v.Nanoseconds(),
	})
}

func (dv *DurationValue) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var uv struct {
		H json.RawMessage `json:"hash"`
		V int64           `json:"value"`
	}

	if err := enc.Unmarshal(b, &uv); err != nil {
		return err
	}

	var err error
	var h valuehash.Hash
	if h, err = valuehash.Decode(enc, uv.H); err != nil {
		return err
	}

	dv.h = h
	dv.v = time.Duration(uv.V)

	return nil
}
