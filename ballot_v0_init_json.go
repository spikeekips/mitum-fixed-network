package mitum

import (
	"encoding/json"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/errors"
	"github.com/spikeekips/mitum/valuehash"
)

type INITBallotV0PackerJSON struct {
	BaseBallotV0PackerJSON
	PB json.RawMessage `json:"previous_block"`
	PR Round           `json:"previous_round"`
	VR interface{}     `json:"voteresult"`
}

func (ib INITBallotV0) PackJSON(enc *encoder.JSONEncoder) (interface{}, error) {
	var jpb json.RawMessage
	if h, err := enc.Marshal(ib.PreviousBlock()); err != nil {
		return nil, err
	} else {
		jpb = h
	}

	bb, err := PackBaseBallotJSON(ib, enc)
	if err != nil {
		return nil, err
	}
	return INITBallotV0PackerJSON{
		BaseBallotV0PackerJSON: bb,
		PB:                     jpb,
		PR:                     ib.PreviousRound(),
		VR:                     ib.VoteResult(),
	}, nil
}

func (ib *INITBallotV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var nib INITBallotV0PackerJSON
	if err := enc.Unmarshal(b, &nib); err != nil {
		return err
	}

	eh, ebh, bb, err := UnpackBaseBallotJSON(nib.BaseBallotV0PackerJSON, enc)
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

	ib.BaseBallotV0 = bb
	ib.h = eh
	ib.bh = ebh
	ib.previousBlock = epb

	return nil
}
