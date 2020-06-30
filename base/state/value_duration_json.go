package state

import (
	"time"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (dv DurationValue) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(struct {
		jsonenc.HintedHead
		H valuehash.Hash `json:"hash"`
		V int64          `json:"value"`
	}{
		HintedHead: jsonenc.NewHintedHead(dv.Hint()),
		H:          dv.Hash(),
		V:          dv.v.Nanoseconds(),
	})
}

func (dv *DurationValue) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uv struct {
		H valuehash.Bytes `json:"hash"`
		V int64           `json:"value"`
	}

	if err := enc.Unmarshal(b, &uv); err != nil {
		return err
	}

	dv.h = uv.H
	dv.v = time.Duration(uv.V)

	return nil
}
