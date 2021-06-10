package ballot // nolint

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type INITV0PackerJSON struct {
	BaseBallotV0PackerJSON
	PB  valuehash.Hash `json:"previous_block"`
	VR  base.Voteproof `json:"voteproof"`
	AVR base.Voteproof `json:"accept_voteproof"`
}

func (ib INITV0) MarshalJSON() ([]byte, error) {
	bb, err := PackBaseBallotV0JSON(ib)
	if err != nil {
		return nil, err
	}
	return jsonenc.Marshal(INITV0PackerJSON{
		BaseBallotV0PackerJSON: bb,
		PB:                     ib.previousBlock,
		VR:                     ib.voteproof,
		AVR:                    ib.acceptVoteproof,
	})
}

type INITV0UnpackerJSON struct {
	BaseBallotV0UnpackerJSON
	PB  valuehash.Bytes `json:"previous_block"`
	VR  json.RawMessage `json:"voteproof"`
	AVR json.RawMessage `json:"accept_voteproof"`
}

func (ib *INITV0) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	bb, bf, err := ib.BaseBallotV0.unpackJSON(b, enc)
	if err != nil {
		return err
	}

	var nib INITV0UnpackerJSON
	if err := enc.Unmarshal(b, &nib); err != nil {
		return err
	}

	return ib.unpack(enc, bb, bf, nib.PB, nib.VR, nib.AVR)
}

type INITFactV0PackerJSON struct {
	BaseBallotFactV0PackerJSON
	PB valuehash.Hash `json:"previous_block"`
}

func (ibf INITFactV0) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(INITFactV0PackerJSON{
		BaseBallotFactV0PackerJSON: NewBaseBallotFactV0PackerJSON(ibf.BaseFactV0, ibf.Hint()),
		PB:                         ibf.previousBlock,
	})
}

type INITFactV0UnpackerJSON struct {
	BaseBallotFactV0PackerJSON
	PB valuehash.Bytes `json:"previous_block"`
}

func (ibf *INITFactV0) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var err error

	var bf BaseFactV0
	if bf, err = ibf.BaseFactV0.unpackJSON(b, enc); err != nil {
		return err
	}

	var ubf INITFactV0UnpackerJSON
	if err := enc.Unmarshal(b, &ubf); err != nil {
		return err
	}

	return ibf.unpack(enc, bf, ubf.PB)
}
