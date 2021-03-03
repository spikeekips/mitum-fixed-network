package ballot

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type ProposalV0PackerJSON struct {
	BaseBallotV0PackerJSON
	SL []valuehash.Hash `json:"seals"`
	VR base.Voteproof   `json:"voteproof"`
}

func (pr ProposalV0) MarshalJSON() ([]byte, error) {
	bb, err := PackBaseBallotV0JSON(pr)
	if err != nil {
		return nil, err
	}

	return jsonenc.Marshal(ProposalV0PackerJSON{
		BaseBallotV0PackerJSON: bb,
		SL:                     pr.seals,
		VR:                     pr.voteproof,
	})
}

type ProposalV0UnpackerJSON struct {
	BaseBallotV0UnpackerJSON
	SL []valuehash.Bytes `json:"seals"`
	VR json.RawMessage   `json:"voteproof"`
}

func (pr *ProposalV0) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	bb, bf, err := pr.BaseBallotV0.unpackJSON(b, enc)
	if err != nil {
		return err
	}

	var npb ProposalV0UnpackerJSON
	if err := enc.Unmarshal(b, &npb); err != nil {
		return err
	}

	seals := make([]valuehash.Hash, len(npb.SL))
	for i := range npb.SL {
		seals[i] = npb.SL[i]
	}

	return pr.unpack(enc, bb, bf, seals, npb.VR)
}
