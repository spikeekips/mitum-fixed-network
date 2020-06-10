package base

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
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

func DecodeAddress(enc encoder.Encoder, b []byte) (Address, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Address); !ok {
		return nil, hint.InvalidTypeError.Errorf("not Address; type=%T", i)
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

func DecodeVoteproof(enc encoder.Encoder, b []byte) (Voteproof, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Voteproof); !ok {
		return nil, hint.InvalidTypeError.Errorf("not Voteproof; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodePolicyOperationBody(enc encoder.Encoder, b []byte) (PolicyOperationBody, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(PolicyOperationBody); !ok {
		return nil, hint.InvalidTypeError.Errorf("not PolicyOperationBody; type=%T", i)
	} else {
		return v, nil
	}
}
