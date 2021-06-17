package operation

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

func DecodeOperation(b []byte, enc encoder.Encoder) (Operation, error) {
	if i, err := enc.Decode(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Operation); !ok {
		return nil, util.WrongTypeError.Errorf("not Fact; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeFactSign(b []byte, enc encoder.Encoder) (FactSign, error) {
	if hinter, err := enc.Decode(b); err != nil {
		return nil, err
	} else if f, ok := hinter.(FactSign); !ok {
		return nil, xerrors.Errorf("not FactSign, %T", hinter)
	} else {
		return f, nil
	}
}

func DecodeReasonError(b []byte, enc encoder.Encoder) (ReasonError, error) {
	if len(b) < 1 {
		return nil, nil
	}

	if hinter, err := enc.Decode(b); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if f, ok := hinter.(ReasonError); !ok {
		return nil, xerrors.Errorf("not ReasonError, %T", hinter)
	} else {
		return f, nil
	}
}
