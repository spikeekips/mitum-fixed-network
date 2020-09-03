package tree

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

func DecodeFixedTree(enc encoder.Encoder, b []byte) (FixedTree, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return FixedTree{}, err
	} else if i == nil {
		return NewFixedTree(nil, nil)
	} else if v, ok := i.(FixedTree); !ok {
		return FixedTree{}, hint.InvalidTypeError.Errorf("not FixedTree; type=%T", i)
	} else {
		return v, nil
	}
}
