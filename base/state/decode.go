package state

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

func DecodeValue(b []byte, enc encoder.Encoder) (Value, error) {
	if i, err := enc.Decode(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Value); !ok {
		return nil, util.WrongTypeError.Errorf("not state.Value; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeState(b []byte, enc encoder.Encoder) (State, error) {
	if i, err := enc.Decode(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(State); !ok {
		return nil, util.WrongTypeError.Errorf("not state.State; type=%T", i)
	} else {
		return v, nil
	}
}
