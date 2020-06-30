package state

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (sv *SliceValue) unpack(enc encoder.Encoder, h valuehash.Hash, bValue [][]byte) error {
	v := make([]hint.Hinter, len(bValue))
	for i, r := range bValue {
		decoded, err := enc.DecodeByHint(r)
		if err != nil {
			return err
		}

		v[i] = decoded
	}

	var b []byte
	if usv, err := (SliceValue{}).set(v); err != nil {
		return err
	} else {
		b = usv.b
	}

	sv.h = h
	sv.b = b
	sv.v = v

	return nil
}
