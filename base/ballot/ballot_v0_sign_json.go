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
	bb, bf, err := sb.BaseBallotV0.unpackJSON(b, enc)
	if err != nil {
		return err
	}

	var nib SIGNBallotV0UnpackerJSON
	if err := enc.Unmarshal(b, &nib); err != nil {
		return err
	}

	return sb.unpack(enc, bb, bf, nib.PR, nib.NB)
}
