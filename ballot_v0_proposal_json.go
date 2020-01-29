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

func (pr ProposalV0) PackJSON(enc *encoder.JSONEncoder) (interface{}, error) {
	var jsl json.RawMessage
	if h, err := enc.Marshal(pr.ProposalV0Fact.Seals()); err != nil {
		return nil, err
	} else {
		jsl = h
	}

	bb, err := PackBaseBallotJSON(pr, enc)
	if err != nil {
		return nil, err
	}
	return ProposalV0PackerJSON{
		BaseBallotV0PackerJSON: bb,
		SL:                     jsl,
	}, nil
}

func (pr *ProposalV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var npb ProposalV0PackerJSON
	if err := enc.Unmarshal(b, &npb); err != nil {
		return err
	}

	eh, ebh, efh, efsg, bb, bf, err := UnpackBaseBallotJSON(npb.BaseBallotV0PackerJSON, enc)
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

	pr.BaseBallotV0 = bb
	pr.h = eh
	pr.bodyHash = ebh
	pr.factHash = efh
	pr.factSignature = efsg
	pr.ProposalV0Fact = ProposalV0Fact{
		BaseBallotV0Fact: bf,
		seals:            esl,
	}

	return nil
}
