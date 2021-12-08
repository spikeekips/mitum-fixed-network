package base

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

func DecodeFact(b []byte, enc encoder.Encoder) (Fact, error) {
	if i, err := enc.Decode(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Fact); !ok {
		return nil, util.WrongTypeError.Errorf("not Fact; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeNode(b []byte, enc encoder.Encoder) (Node, error) {
	if i, err := enc.Decode(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Node); !ok {
		return nil, util.WrongTypeError.Errorf("not Node; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeVoteproof(b []byte, enc encoder.Encoder) (Voteproof, error) {
	if i, err := enc.Decode(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Voteproof); !ok {
		return nil, util.WrongTypeError.Errorf("not Voteproof; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeSignedBallotFact(b []byte, enc encoder.Encoder) (SignedBallotFact, error) {
	if i, err := enc.Decode(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(SignedBallotFact); !ok {
		return nil, util.WrongTypeError.Errorf("not SignedBallotFact; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeFactSign(b []byte, enc encoder.Encoder) (FactSign, error) {
	if hinter, err := enc.Decode(b); err != nil {
		return nil, err
	} else if f, ok := hinter.(FactSign); !ok {
		return nil, errors.Errorf("not FactSign, %T", hinter)
	} else {
		return f, nil
	}
}

func DecodeBallot(b []byte, enc encoder.Encoder) (Ballot, error) {
	ht, err := enc.Decode(b)
	if err != nil {
		return nil, err
	}

	if ht == nil {
		return nil, nil
	}

	bl, ok := ht.(Ballot)
	if !ok {
		return nil, util.WrongTypeError.Errorf("not Ballot; type=%T", ht)
	}

	return bl, nil
}

func DecodeBallotFact(b []byte, enc encoder.Encoder) (BallotFact, error) {
	if i, err := enc.Decode(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(BallotFact); !ok {
		return nil, util.WrongTypeError.Errorf("not BallotFact; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeBallotFactSign(b []byte, enc encoder.Encoder) (BallotFactSign, error) {
	if hinter, err := enc.Decode(b); err != nil {
		return nil, err
	} else if f, ok := hinter.(BallotFactSign); !ok {
		return nil, errors.Errorf("not BallotFactSign, %T", hinter)
	} else {
		return f, nil
	}
}

func DecodeProposalFact(b []byte, enc encoder.Encoder) (ProposalFact, error) {
	ht, err := enc.Decode(b)
	if err != nil {
		return nil, err
	}

	if ht == nil {
		return nil, nil
	}

	fact, ok := ht.(ProposalFact)
	if !ok {
		return nil, util.WrongTypeError.Errorf("not ProposalFact; type=%T", ht)
	}

	return fact, nil
}

func DecodeProposal(b []byte, enc encoder.Encoder) (Proposal, error) {
	i, err := DecodeBallot(b, enc)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Proposal: %w", err)
	}

	j, ok := i.(Proposal)
	if !ok {
		return nil, errors.Errorf("not Proposal, %T", i)
	}

	return j, nil
}
