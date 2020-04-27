package state

import (
	"encoding/json"
	"time"

	"github.com/spikeekips/mitum/base/valuehash"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
)

func (dv DurationValue) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(struct {
		jsonencoder.HintedHead
		H valuehash.Hash `json:"hash"`
		V int64          `json:"value"`
	}{
		HintedHead: jsonencoder.NewHintedHead(dv.Hint()),
		H:          dv.Hash(),
		V:          dv.v.Nanoseconds(),
	})
}

func (dv *DurationValue) UnpackJSON(b []byte, enc *jsonencoder.Encoder) error {
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
