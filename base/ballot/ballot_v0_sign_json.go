package ballot

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

type SIGNBallotV0PackerJSON struct {
	BaseBallotV0PackerJSON
	PR valuehash.Hash `json:"proposal"`
	NB valuehash.Hash `json:"new_block"`
}

func (sb SIGNBallotV0) MarshalJSON() ([]byte, error) {
	bb, err := PackBaseBallotV0JSON(sb)
	if err != nil {
		return nil, err
	}

	return util.JSONMarshal(SIGNBallotV0PackerJSON{
		BaseBallotV0PackerJSON: bb,
		PR:                     sb.proposal,
		NB:                     sb.newBlock,
	})
}

type SIGNBallotV0UnpackerJSON struct {
	BaseBallotV0UnpackerJSON
	PR json.RawMessage `json:"proposal"`
	NB json.RawMessage `json:"new_block"`
}

func (sb *SIGNBallotV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error { // nolint
	var nib SIGNBallotV0UnpackerJSON
	if err := enc.Unmarshal(b, &nib); err != nil {
		return err
	} else if err := sb.Hint().IsCompatible(nib.JSONPackHintedHead.H); err != nil {
		return err
	}

	bb, bf, err := UnpackBaseBallotV0JSON(nib.BaseBallotV0UnpackerJSON, enc)
	if err != nil {
		return err
	}

	var epr, enb valuehash.Hash
	if i, err := valuehash.Decode(enc, nib.PR); err != nil {
		return err
	} else {
		epr = i
	}

	if i, err := valuehash.Decode(enc, nib.NB); err != nil {
		return err
	} else {
		enb = i
	}

	sb.BaseBallotV0 = bb
	sb.SIGNBallotFactV0 = SIGNBallotFactV0{
		BaseBallotFactV0: bf,
		proposal:         epr,
		newBlock:         enb,
	}

	return nil
}
