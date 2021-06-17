package base

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
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

func DecodeVoteproofNodeFact(b []byte, enc encoder.Encoder) (VoteproofNodeFact, error) {
	var vp VoteproofNodeFact
	if i, err := enc.Decode(b); err != nil {
		return vp, err
	} else if i == nil {
		return vp, nil
	} else if v, ok := i.(VoteproofNodeFact); !ok {
		return vp, util.WrongTypeError.Errorf("not VoteproofNodeFact; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeAddressFromString(s string, enc encoder.Encoder) (Address, error) {
	hs, err := hint.ParseHintedString(s)
	if err != nil {
		return nil, err
	}

	kd := encoder.NewHintedString(hs.Hint(), hs.Body())
	if k, err := kd.Decode(enc); err != nil {
		return nil, err
	} else if a, ok := k.(Address); !ok {
		return nil, util.WrongTypeError.Errorf("not Address; type=%T", k)
	} else {
		return a, nil
	}
}
