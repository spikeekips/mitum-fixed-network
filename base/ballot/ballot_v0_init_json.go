package ballot

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

type INITBallotV0PackerJSON struct {
	BaseBallotV0PackerJSON
	PB valuehash.Hash `json:"previous_block"`
	PR base.Round     `json:"previous_round"`
	VR base.Voteproof `json:"voteproof"`
}

func (ib INITBallotV0) MarshalJSON() ([]byte, error) {
	bb, err := PackBaseBallotV0JSON(ib)
	if err != nil {
		return nil, err
	}
	return util.JSONMarshal(INITBallotV0PackerJSON{
		BaseBallotV0PackerJSON: bb,
		PB:                     ib.previousBlock,
		PR:                     ib.previousRound,
		VR:                     ib.voteproof,
	})
}

type INITBallotV0UnpackerJSON struct {
	BaseBallotV0UnpackerJSON
	PB json.RawMessage `json:"previous_block"`
	PR base.Round      `json:"previous_round"`
	VR json.RawMessage `json:"voteproof"`
}

func (ib *INITBallotV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var nib INITBallotV0UnpackerJSON
	if err := enc.Unmarshal(b, &nib); err != nil {
		return err
	} else if err := ib.Hint().IsCompatible(nib.JSONPackHintedHead.H); err != nil {
		return err
	}

	bb, bf, err := UnpackBaseBallotV0JSON(nib.BaseBallotV0UnpackerJSON, enc)
	if err != nil {
		return err
	}

	// previousblock
	var epb valuehash.Hash
	if i, err := valuehash.Decode(enc, nib.PB); err != nil {
		return err
	} else {
		epb = i
	}

	var voteproof base.Voteproof
	if i, err := base.DecodeVoteproof(enc, nib.VR); err != nil {
		return err
	} else {
		voteproof = i
	}

	ib.BaseBallotV0 = bb
	ib.INITBallotFactV0 = INITBallotFactV0{
		BaseBallotFactV0: bf,
		previousBlock:    epb,
		previousRound:    nib.PR,
	}
	ib.voteproof = voteproof

	return nil
}

type INITBallotFactV0PackerJSON struct {
	BaseBallotFactV0PackerJSON
	PB valuehash.Hash `json:"previous_block"`
	PR base.Round     `json:"previous_round"`
}

func (ibf INITBallotFactV0) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(INITBallotFactV0PackerJSON{
		BaseBallotFactV0PackerJSON: NewBaseBallotFactV0PackerJSON(ibf.BaseBallotFactV0, ibf.Hint()),
		PB:                         ibf.previousBlock,
		PR:                         ibf.previousRound,
	})
}

type INITBallotFactV0UnpackerJSON struct {
	BaseBallotFactV0PackerJSON
	PB json.RawMessage `json:"previous_block"`
	PR base.Round      `json:"previous_round"`
}

func (ibf *INITBallotFactV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var ubf INITBallotFactV0UnpackerJSON
	if err := enc.Unmarshal(b, &ubf); err != nil {
		return err
	}

	var err error
	var pb valuehash.Hash
	if pb, err = valuehash.Decode(enc, ubf.PB); err != nil {
		return err
	}

	ibf.BaseBallotFactV0.height = ubf.HT
	ibf.BaseBallotFactV0.round = ubf.RD
	ibf.previousBlock = pb
	ibf.previousRound = ubf.PR

	return nil
}
