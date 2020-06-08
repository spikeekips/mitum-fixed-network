package ballot

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base/valuehash"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
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
	OP []json.RawMessage `json:"operations"`
	SL []json.RawMessage `json:"seals"`
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

	ops := make([][]byte, len(npb.OP))
	for i, r := range npb.OP {
		ops[i] = r
	}

	seals := make([][]byte, len(npb.SL))
	for i, r := range npb.SL {
		seals[i] = r
	}

	return pr.unpack(enc, bb, bf, ops, seals)
}
