package mitum

import (
	"encoding/json"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/errors"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

type ProposalV0PackerJSON struct {
	BaseBallotV0PackerJSON
	SL []valuehash.Hash `json:"seals"`
}

func (pr ProposalV0) MarshalJSON() ([]byte, error) {
	bb, err := PackBaseBallotV0JSON(pr)
	if err != nil {
		return nil, err
	}

	return util.JSONMarshal(ProposalV0PackerJSON{
		BaseBallotV0PackerJSON: bb,
		SL:                     pr.seals,
	})
}

type ProposalV0UnpackerJSON struct {
	BaseBallotV0UnpackerJSON
	SL json.RawMessage `json:"seals"`
}

func (pr *ProposalV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var npb ProposalV0UnpackerJSON
	if err := enc.Unmarshal(b, &npb); err != nil {
		return err
	} else if err := pr.Hint().IsCompatible(npb.JSONPackHintedHead.H); err != nil {
		return err
	}

	eh, ebh, efh, efsg, bb, bf, err := UnpackBaseBallotV0JSON(npb.BaseBallotV0UnpackerJSON, enc)
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
	pr.ProposalFactV0 = ProposalFactV0{
		BaseBallotFactV0: bf,
		seals:            esl,
	}

	return nil
}
