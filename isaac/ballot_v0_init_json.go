package isaac

import (
	"encoding/json"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/errors"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

type INITBallotV0PackerJSON struct {
	BaseBallotV0PackerJSON
	PB valuehash.Hash `json:"previous_block"`
	PR Round          `json:"previous_round"`
	VR VoteProof      `json:"voteproof"`
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
		VR:                     ib.voteProof,
	})
}

type INITBallotV0UnpackerJSON struct {
	BaseBallotV0UnpackerJSON
	PB json.RawMessage `json:"previous_block"`
	PR Round           `json:"previous_round"`
	VR json.RawMessage `json:"voteproof"`
}

func (ib *INITBallotV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var nib INITBallotV0UnpackerJSON
	if err := enc.Unmarshal(b, &nib); err != nil {
		return err
	} else if err := ib.Hint().IsCompatible(nib.JSONPackHintedHead.H); err != nil {
		return err
	}

	eh, ebh, efh, efsg, bb, bf, err := UnpackBaseBallotV0JSON(nib.BaseBallotV0UnpackerJSON, enc)
	if err != nil {
		return err
	}

	// previousblock
	var epb valuehash.Hash
	if i, err := enc.DecodeByHint(nib.PB); err != nil {
		return err
	} else if v, ok := i.(valuehash.Hash); !ok {
		return errors.InvalidTypeError.Wrapf("not valuehash.Hash; type=%T", i)
	} else {
		epb = v
	}

	var vp VoteProof
	if i, err := decodeVoteProof(enc, nib.VR); err != nil {
		return err
	} else {
		vp = i
	}

	ib.BaseBallotV0 = bb
	ib.h = eh
	ib.bodyHash = ebh
	ib.factHash = efh
	ib.factSignature = efsg
	ib.INITBallotFactV0 = INITBallotFactV0{
		BaseBallotFactV0: bf,
		previousBlock:    epb,
		previousRound:    nib.PR,
	}
	ib.voteProof = vp

	return nil
}

type INITBallotFactV0PackerJSON struct {
	BaseBallotFactV0PackerJSON
	PB valuehash.Hash `json:"previous_block"`
	PR Round          `json:"previous_round"`
}

func (ibf INITBallotFactV0) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(INITBallotFactV0PackerJSON{
		BaseBallotFactV0PackerJSON: NewBaseBallotFactV0PackerJSON(ibf.BaseBallotFactV0),
		PB:                         ibf.previousBlock,
		PR:                         ibf.previousRound,
	})
}
