package state

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

func DecodeValue(enc encoder.Encoder, b []byte) (Value, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Value); !ok {
		return nil, util.WrongTypeError.Errorf("not state.Value; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeState(enc encoder.Encoder, b []byte) (State, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(State); !ok {
		return nil, util.WrongTypeError.Errorf("not state.State; type=%T", i)
	} else {
		return v, nil
	}
}
