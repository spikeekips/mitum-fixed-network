package isaac

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

type ProposalV0PackerJSON struct {
	BaseBallotV0PackerJSON
	OP []valuehash.Hash `json:"operations"`
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
		OP:                     pr.operations,
	})
}

type ProposalV0UnpackerJSON struct {
	BaseBallotV0UnpackerJSON
	OP []json.RawMessage `json:"operations"`
	SL []json.RawMessage `json:"seals"`
}

func (pr *ProposalV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var npb ProposalV0UnpackerJSON
	if err := enc.Unmarshal(b, &npb); err != nil {
		return err
	} else if err := pr.Hint().IsCompatible(npb.JSONPackHintedHead.H); err != nil {
		return err
	}

	bb, bf, err := UnpackBaseBallotV0JSON(npb.BaseBallotV0UnpackerJSON, enc)
	if err != nil {
		return err
	}

	var ol, sl []valuehash.Hash
	for _, r := range npb.OP {
		if h, err := valuehash.Decode(enc, r); err != nil {
			return err
		} else {
			ol = append(ol, h)
		}
	}

	for _, r := range npb.SL {
		if h, err := valuehash.Decode(enc, r); err != nil {
			return err
		} else {
			sl = append(sl, h)
		}
	}

	pr.BaseBallotV0 = bb
	pr.ProposalFactV0 = ProposalFactV0{
		BaseBallotFactV0: bf,
		operations:       ol,
		seals:            sl,
	}

	return nil
}
