package operation

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

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

func DecodeFactSign(enc encoder.Encoder, b []byte) (FactSign, error) {
	if hinter, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if f, ok := hinter.(FactSign); !ok {
		return nil, xerrors.Errorf("not FactSign, %T", hinter)
	} else {
		return f, nil
	}
}

func DecodeOperationInfo(enc encoder.Encoder, b []byte) (OperationInfo, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(OperationInfo); !ok {
		return nil, hint.InvalidTypeError.Errorf("not state.OperationInfo; type=%T", i)
	} else {
		return v, nil
	}
}
