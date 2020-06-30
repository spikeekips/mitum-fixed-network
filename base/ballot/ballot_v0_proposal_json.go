package ballot

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type ProposalV0PackerJSON struct {
	BaseBallotV0PackerJSON
	OP []valuehash.Hash `json:"operations"`
	SL []valuehash.Hash `json:"seals"`
}

func (pr ProposalV0) MarshalJSON() ([]byte, error) {
	bb, err := PackBaseBallotV0JSON(pr)
	if err != nil {
		return nil, err
	}

	return jsonenc.Marshal(ProposalV0PackerJSON{
		BaseBallotV0PackerJSON: bb,
		SL:                     pr.seals,
		OP:                     pr.operations,
	})
}

type ProposalV0UnpackerJSON struct {
	BaseBallotV0UnpackerJSON
	OP []valuehash.Bytes `json:"operations"`
	SL []valuehash.Bytes `json:"seals"`
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

	ops := make([]valuehash.Hash, len(npb.OP))
	for i := range npb.OP {
		ops[i] = npb.OP[i]
	}

	seals := make([]valuehash.Hash, len(npb.SL))
	for i := range npb.SL {
		seals[i] = npb.SL[i]
	}

	return pr.unpack(enc, bb, bf, ops, seals)
}
