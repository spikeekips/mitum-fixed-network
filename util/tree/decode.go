package tree

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

func DecodeFixedTreeNode(enc encoder.Encoder, b []byte) (FixedTreeNode, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(FixedTreeNode); !ok {
		return nil, hint.InvalidTypeError.Errorf("not FixedTree; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeFixedTree(enc encoder.Encoder, b []byte) (FixedTree, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return FixedTree{}, err
	} else if i == nil {
		return NewFixedTree(nil), nil
	} else if v, ok := i.(FixedTree); !ok {
		return FixedTree{}, hint.InvalidTypeError.Errorf("not FixedTree; type=%T", i)
	} else {
		return v, nil
	}
}
