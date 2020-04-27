package ballot // nolint

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/valuehash"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
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

	return jsonencoder.Marshal(ACCEPTBallotV0PackerJSON{
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

func (ab *ACCEPTBallotV0) UnpackJSON(b []byte, enc *jsonencoder.Encoder) error { // nolint
	bb, bf, err := ab.BaseBallotV0.unpackJSON(b, enc)
	if err != nil {
		return err
	}

	var nab ACCEPTBallotV0UnpackerJSON
	if err := enc.Unmarshal(b, &nab); err != nil {
		return err
	}

	return ab.unpack(enc, bb, bf, nab.PR, nab.NB, nab.VR)
}

type ACCEPTBallotFactV0PackerJSON struct {
	BaseBallotFactV0PackerJSON
	PR valuehash.Hash `json:"proposal"`
	NB valuehash.Hash `json:"new_block"`
}

func (abf ACCEPTBallotFactV0) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(ACCEPTBallotFactV0PackerJSON{
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

func (abf *ACCEPTBallotFactV0) UnpackJSON(b []byte, enc *jsonencoder.Encoder) error {
	var err error

	var bf BaseBallotFactV0
	if bf, err = abf.BaseBallotFactV0.unpackJSON(b, enc); err != nil {
		return err
	}

	var ubf ACCEPTBallotFactV0UnpackerJSON
	if err := enc.Unmarshal(b, &ubf); err != nil {
		return err
	}

	return abf.unpack(enc, bf, ubf.PR, ubf.NB)
}
