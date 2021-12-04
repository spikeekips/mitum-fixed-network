package state

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (sv *SliceValue) unpack(enc encoder.Encoder, h valuehash.Hash, bValue [][]byte) error {
	v := make([]hint.Hinter, len(bValue))
	for i, r := range bValue {
		decoded, err := enc.Decode(r)
		if err != nil {
			return err
		}

		v[i] = decoded
	}

	usv, err := (SliceValueHinter).set(v)
	if err != nil {
		return err
	}

	sv.h = h
	sv.b = usv.b
	sv.v = v

	return nil
}
