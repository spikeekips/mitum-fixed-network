package ballot

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

func Decode(enc encoder.Encoder, b []byte) (Ballot, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Ballot); !ok {
		return nil, hint.InvalidTypeError.Errorf("not Ballot; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeProposal(enc encoder.Encoder, b []byte) (Proposal, error) {
	if i, err := Decode(enc, b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Proposal); !ok {
		return nil, hint.InvalidTypeError.Errorf("not Proposal; type=%T", i)
	} else {
		return v, nil
	}
}
