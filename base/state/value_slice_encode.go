package state

import (
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

func (sv *SliceValue) unpack(enc encoder.Encoder, bHash []byte, bValue [][]byte) error {
	var h valuehash.Hash
	if i, err := valuehash.Decode(enc, bHash); err != nil {
		return err
	} else {
		h = i
	}

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
