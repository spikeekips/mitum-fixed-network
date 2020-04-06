package operation

import (
	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
)

func DecodeFact(enc encoder.Encoder, b []byte) (Fact, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Fact); !ok {
		return nil, hint.InvalidTypeError.Errorf("not Fact; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeOperation(enc encoder.Encoder, b []byte) (Operation, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Operation); !ok {
		return nil, hint.InvalidTypeError.Errorf("not Fact; type=%T", i)
	} else {
		return v, nil
	}
}
