package mitum

import (
	"encoding/json"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/errors"
	"github.com/spikeekips/mitum/valuehash"
)

type ProposalV0PackerJSON struct {
	BaseBallotV0PackerJSON
	SL json.RawMessage `json:"seals"`
}

func (pb ProposalV0) PackJSON(enc *encoder.JSONEncoder) (interface{}, error) {
	var jsl json.RawMessage
	if h, err := enc.Marshal(pb.Seals()); err != nil {
		return nil, err
	} else {
		jsl = h
	}

	bb, err := PackBaseBallotJSON(pb, enc)
	if err != nil {
		return nil, err
	}
	return ProposalV0PackerJSON{
		BaseBallotV0PackerJSON: bb,
		SL:                     jsl,
	}, nil
}

func (pb *ProposalV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var npb ProposalV0PackerJSON
	if err := enc.Unmarshal(b, &npb); err != nil {
		return err
	}

	eh, ebh, bb, err := UnpackBaseBallotJSON(npb.BaseBallotV0PackerJSON, enc)
	if err != nil {
		return err
	}

	var sl []json.RawMessage
	if err := enc.Unmarshal(npb.SL, &sl); err != nil {
		return err
	}

	var esl []valuehash.Hash
	for _, r := range sl {
		if i, err := enc.DecodeByHint(r); err != nil {
			return err
		} else if v, ok := i.(valuehash.Hash); !ok {
			return errors.InvalidTypeError.Wrapf("not valuehash.Hash; type=%T", i)
		} else {
			esl = append(esl, v)
		}
	}

	pb.BaseBallotV0 = bb
	pb.h = eh
	pb.bh = ebh
	pb.seals = esl

	return nil
}
