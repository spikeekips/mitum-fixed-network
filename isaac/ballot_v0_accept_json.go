package isaac

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

type ACCEPTBallotV0PackerJSON struct {
	BaseBallotV0PackerJSON
	PR valuehash.Hash `json:"proposal"`
	NB valuehash.Hash `json:"new_block"`
	VR base.Voteproof `json:"voteproof"`
}

func (ab ACCEPTBallotV0) MarshalJSON() ([]byte, error) {
	bb, err := PackBaseBallotV0JSON(ab)
	if err != nil {
		return nil, err
	}

	return util.JSONMarshal(ACCEPTBallotV0PackerJSON{
		BaseBallotV0PackerJSON: bb,
		PR:                     ab.proposal,
		NB:                     ab.newBlock,
		VR:                     ab.voteproof,
	})
}

type ACCEPTBallotV0UnpackerJSON struct {
	BaseBallotV0UnpackerJSON
	PR json.RawMessage `json:"proposal"`
	NB json.RawMessage `json:"new_block"`
	VR json.RawMessage `json:"voteproof"`
}

func (ab *ACCEPTBallotV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error { // nolint
	var nab ACCEPTBallotV0UnpackerJSON
	if err := enc.Unmarshal(b, &nab); err != nil {
		return err
	} else if err := ab.Hint().IsCompatible(nab.JSONPackHintedHead.H); err != nil {
		return err
	}

	bb, bf, err := UnpackBaseBallotV0JSON(nab.BaseBallotV0UnpackerJSON, enc)
	if err != nil {
		return err
	}

	var epr, enb valuehash.Hash
	if i, err := valuehash.Decode(enc, nab.PR); err != nil {
		return err
	} else {
		epr = i
	}

	if i, err := valuehash.Decode(enc, nab.NB); err != nil {
		return err
	} else {
		enb = i
	}

	var voteproof base.Voteproof
	if i, err := base.DecodeVoteproof(enc, nab.VR); err != nil {
		return err
	} else {
		voteproof = i
	}

	ab.BaseBallotV0 = bb
	ab.ACCEPTBallotFactV0 = ACCEPTBallotFactV0{
		BaseBallotFactV0: bf,
		proposal:         epr,
		newBlock:         enb,
	}
	ab.voteproof = voteproof

	return nil
}

type ACCEPTBallotFactV0PackerJSON struct {
	BaseBallotFactV0PackerJSON
	PR valuehash.Hash `json:"proposal"`
	NB valuehash.Hash `json:"new_block"`
}

func (abf ACCEPTBallotFactV0) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(ACCEPTBallotFactV0PackerJSON{
		BaseBallotFactV0PackerJSON: NewBaseBallotFactV0PackerJSON(abf.BaseBallotFactV0, abf.Hint()),
		PR:                         abf.proposal,
		NB:                         abf.newBlock,
	})
}

type ACCEPTBallotFactV0UnpackerJSON struct {
	BaseBallotFactV0PackerJSON
	PR json.RawMessage `json:"proposal"`
	NB json.RawMessage `json:"new_block"`
}

func (abf *ACCEPTBallotFactV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var ubf ACCEPTBallotFactV0UnpackerJSON
	if err := enc.Unmarshal(b, &ubf); err != nil {
		return err
	}

	var err error
	var pr, nb valuehash.Hash
	if pr, err = valuehash.Decode(enc, ubf.PR); err != nil {
		return err
	}
	if nb, err = valuehash.Decode(enc, ubf.NB); err != nil {
		return err
	}

	abf.BaseBallotFactV0.height = ubf.HT
	abf.BaseBallotFactV0.round = ubf.RD
	abf.proposal = pr
	abf.newBlock = nb

	return nil
}
