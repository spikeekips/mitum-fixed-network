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

func DecodeReasonError(enc encoder.Encoder, b []byte) (ReasonError, error) {
	if len(b) < 1 {
		return nil, nil
	}
	if hinter, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if f, ok := hinter.(ReasonError); !ok {
		return nil, xerrors.Errorf("not ReasonError, %T", hinter)
	} else {
		return f, nil
	}
}
