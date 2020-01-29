package mitum

import (
	"encoding/json"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/errors"
	"github.com/spikeekips/mitum/valuehash"
)

type SIGNBallotV0PackerJSON struct {
	BaseBallotV0PackerJSON
	PR json.RawMessage `json:"proposal"`
	NB json.RawMessage `json:"previous_block"`
}

func (sb SIGNBallotV0) PackJSON(enc *encoder.JSONEncoder) (interface{}, error) {
	var jpr, jnb json.RawMessage
	if h, err := enc.Marshal(sb.SIGNBallotV0Fact.proposal); err != nil {
		return nil, err
	} else {
		jpr = h
	}

	if h, err := enc.Marshal(sb.SIGNBallotV0Fact.newBlock); err != nil {
		return nil, err
	} else {
		jnb = h
	}

	bb, err := PackBaseBallotJSON(sb, enc)
	if err != nil {
		return nil, err
	}
	return SIGNBallotV0PackerJSON{
		BaseBallotV0PackerJSON: bb,
		PR:                     jpr,
		NB:                     jnb,
	}, nil
}

func (sb *SIGNBallotV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error { // nolint
	var nib SIGNBallotV0PackerJSON
	if err := enc.Unmarshal(b, &nib); err != nil {
		return err
	}

	eh, ebh, efh, efsg, bb, bf, err := UnpackBaseBallotJSON(nib.BaseBallotV0PackerJSON, enc)
	if err != nil {
		return err
	}

	var epr, enb valuehash.Hash
	if i, err := enc.DecodeByHint(nib.PR); err != nil {
		return err
	} else if v, ok := i.(valuehash.Hash); !ok {
		return errors.InvalidTypeError.Wrapf("not valuehash.Hash; type=%T", i)
	} else {
		epr = v
	}

	if i, err := enc.DecodeByHint(nib.NB); err != nil {
		return err
	} else if v, ok := i.(valuehash.Hash); !ok {
		return errors.InvalidTypeError.Wrapf("not valuehash.Hash; type=%T", i)
	} else {
		enb = v
	}

	sb.BaseBallotV0 = bb
	sb.h = eh
	sb.bodyHash = ebh
	sb.factHash = efh
	sb.factSignature = efsg
	sb.SIGNBallotV0Fact = SIGNBallotV0Fact{
		BaseBallotV0Fact: bf,
		proposal:         epr,
		newBlock:         enb,
	}

	return nil
}
