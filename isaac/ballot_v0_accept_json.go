package isaac

import (
	"encoding/json"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/errors"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

type ACCEPTBallotV0PackerJSON struct {
	BaseBallotV0PackerJSON
	PR valuehash.Hash `json:"proposal"`
	NB valuehash.Hash `json:"new_block"`
	VR Voteproof      `json:"voteproof"`
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

	eh, ebh, efh, efsg, bb, bf, err := UnpackBaseBallotV0JSON(nab.BaseBallotV0UnpackerJSON, enc)
	if err != nil {
		return err
	}

	// TODO use decodehash
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

	var vp Voteproof
	if i, err := decodeVoteproof(enc, nab.VR); err != nil {
		return err
	} else {
		vp = i
	}

	ab.BaseBallotV0 = bb
	ab.h = eh
	ab.bodyHash = ebh
	ab.factHash = efh
	ab.factSignature = efsg
	ab.ACCEPTBallotFactV0 = ACCEPTBallotFactV0{
		BaseBallotFactV0: bf,
		proposal:         epr,
		newBlock:         enb,
	}
	ab.voteproof = vp

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
