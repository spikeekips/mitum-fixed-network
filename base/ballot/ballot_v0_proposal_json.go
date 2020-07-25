package ballot

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type ProposalV0PackerJSON struct {
	BaseBallotV0PackerJSON
	FS []valuehash.Hash `json:"facts"`
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
		FS:                     pr.facts,
	})
}

type ProposalV0UnpackerJSON struct {
	BaseBallotV0UnpackerJSON
	FS []valuehash.Bytes `json:"facts"`
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

	fs := make([]valuehash.Hash, len(npb.FS))
	for i := range npb.FS {
		fs[i] = npb.FS[i]
	}

	seals := make([]valuehash.Hash, len(npb.SL))
	for i := range npb.SL {
		seals[i] = npb.SL[i]
	}

	return pr.unpack(enc, bb, bf, fs, seals)
}
