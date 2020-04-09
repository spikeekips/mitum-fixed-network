package tree

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

func DecodeAVLTree(enc encoder.Encoder, b []byte) (AVLTree, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return AVLTree{}, err
	} else if i == nil {
		return AVLTree{}, nil
	} else if v, ok := i.(AVLTree); !ok {
		return AVLTree{}, hint.InvalidTypeError.Errorf("not AVLTree; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeNode(enc encoder.Encoder, b []byte) (Node, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Node); !ok {
		return nil, hint.InvalidTypeError.Errorf("not Node; type=%T", i)
	} else {
		return v, nil
	}
}
