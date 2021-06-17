package ballot

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

func Decode(b []byte, enc encoder.Encoder) (Ballot, error) {
	if i, err := enc.Decode(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Ballot); !ok {
		return nil, util.WrongTypeError.Errorf("not Ballot; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeProposal(b []byte, enc encoder.Encoder) (Proposal, error) {
	if i, err := Decode(b, enc); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Proposal); !ok {
		return nil, util.WrongTypeError.Errorf("not Proposal; type=%T", i)
	} else {
		return v, nil
	}
}
