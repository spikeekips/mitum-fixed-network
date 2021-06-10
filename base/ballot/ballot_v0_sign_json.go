package ballot

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type SIGNV0PackerJSON struct {
	BaseBallotV0PackerJSON
	PR valuehash.Hash `json:"proposal"`
	NB valuehash.Hash `json:"new_block"`
}

func (sb SIGNV0) MarshalJSON() ([]byte, error) {
	bb, err := PackBaseBallotV0JSON(sb)
	if err != nil {
		return nil, err
	}

	return jsonenc.Marshal(SIGNV0PackerJSON{
		BaseBallotV0PackerJSON: bb,
		PR:                     sb.proposal,
		NB:                     sb.newBlock,
	})
}

type SIGNV0UnpackerJSON struct {
	BaseBallotV0UnpackerJSON
	PR valuehash.Bytes `json:"proposal"`
	NB valuehash.Bytes `json:"new_block"`
}

func (sb *SIGNV0) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	bb, bf, err := sb.BaseBallotV0.unpackJSON(b, enc)
	if err != nil {
		return err
	}

	var nib SIGNV0UnpackerJSON
	if err := enc.Unmarshal(b, &nib); err != nil {
		return err
	}

	return sb.unpack(enc, bb, bf, nib.PR, nib.NB)
}
