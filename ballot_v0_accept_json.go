package mitum

import (
	"encoding/json"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/errors"
	"github.com/spikeekips/mitum/valuehash"
)

type ACCEPTBallotV0PackerJSON struct {
	BaseBallotV0PackerJSON
	PR json.RawMessage `json:"proposal"`
	NB json.RawMessage `json:"previous_block"`
	VR VoteProof       `json:"voteproof"`
}

func (ab ACCEPTBallotV0) PackJSON(enc *encoder.JSONEncoder) (interface{}, error) {
	var jpr, jnb json.RawMessage
	if h, err := enc.Marshal(ab.ACCEPTBallotV0Fact.proposal); err != nil {
		return nil, err
	} else {
		jpr = h
	}

	if h, err := enc.Marshal(ab.ACCEPTBallotV0Fact.newBlock); err != nil {
		return nil, err
	} else {
		jnb = h
	}

	bb, err := PackBaseBallotJSON(ab, enc)
	if err != nil {
		return nil, err
	}
	return ACCEPTBallotV0PackerJSON{
		BaseBallotV0PackerJSON: bb,
		PR:                     jpr,
		NB:                     jnb,
		VR:                     ab.VoteProof(),
	}, nil
}

func (ab *ACCEPTBallotV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error { // nolint
	var nab ACCEPTBallotV0PackerJSON
	if err := enc.Unmarshal(b, &nab); err != nil {
		return err
	}

	eh, ebh, efh, efsg, bb, bf, err := UnpackBaseBallotJSON(nab.BaseBallotV0PackerJSON, enc)
	if err != nil {
		return err
	}

	var epr, enb valuehash.Hash
	if i, err := enc.DecodeByHint(nab.PR); err != nil {
		return err
	} else if v, ok := i.(valuehash.Hash); !ok {
		return errors.InvalidTypeError.Wrapf("not valuehash.Hash; type=%T", i)
	} else {
		epr = v
	}

	if i, err := enc.DecodeByHint(nab.NB); err != nil {
		return err
	} else if v, ok := i.(valuehash.Hash); !ok {
		return errors.InvalidTypeError.Wrapf("not valuehash.Hash; type=%T", i)
	} else {
		enb = v
	}

	ab.BaseBallotV0 = bb
	ab.h = eh
	ab.bodyHash = ebh
	ab.factHash = efh
	ab.factSignature = efsg
	ab.ACCEPTBallotV0Fact = ACCEPTBallotV0Fact{
		BaseBallotV0Fact: bf,
		proposal:         epr,
		newBlock:         enb,
	}

	return nil
}
