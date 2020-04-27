package ballot // nolint

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/valuehash"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
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
	return jsonencoder.Marshal(INITBallotV0PackerJSON{
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

func (ib *INITBallotV0) UnpackJSON(b []byte, enc *jsonencoder.Encoder) error {
	bb, bf, err := ib.BaseBallotV0.unpackJSON(b, enc)
	if err != nil {
		return err
	}

	var nib INITBallotV0UnpackerJSON
	if err := enc.Unmarshal(b, &nib); err != nil {
		return err
	}

	return ib.unpack(enc, bb, bf, nib.PB, nib.PR, nib.VR)
}

type INITBallotFactV0PackerJSON struct {
	BaseBallotFactV0PackerJSON
	PB valuehash.Hash `json:"previous_block"`
	PR base.Round     `json:"previous_round"`
}

func (ibf INITBallotFactV0) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(INITBallotFactV0PackerJSON{
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

func (ibf *INITBallotFactV0) UnpackJSON(b []byte, enc *jsonencoder.Encoder) error {
	var err error

	var bf BaseBallotFactV0
	if bf, err = ibf.BaseBallotFactV0.unpackJSON(b, enc); err != nil {
		return err
	}

	var ubf INITBallotFactV0UnpackerJSON
	if err := enc.Unmarshal(b, &ubf); err != nil {
		return err
	}

	return ibf.unpack(enc, bf, ubf.PB, ubf.PR)
}
