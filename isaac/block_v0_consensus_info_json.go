package isaac

import (
	"encoding/json"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/util"
)

type BlockConsensusInfoV0PackJSON struct {
	encoder.JSONPackHintedHead
	IV Voteproof `json:"init_voteproof,omitempty"`
	AV Voteproof `json:"accept_voteproof,omitempty"`
}

func (bc BlockConsensusInfoV0) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(BlockConsensusInfoV0PackJSON{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(bc.Hint()),
		IV:                 bc.initVoteproof,
		AV:                 bc.acceptVoteproof,
	})
}

type BlockConsensusInfoV0UnpackJSON struct {
	IV json.RawMessage `json:"init_voteproof"`
	AV json.RawMessage `json:"accept_voteproof"`
}

func (bc *BlockConsensusInfoV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var nbc BlockConsensusInfoV0UnpackJSON
	if err := enc.Unmarshal(b, &nbc); err != nil {
		return err
	}

	var err error
	var iv, av Voteproof
	if nbc.IV != nil {
		if iv, err = decodeVoteproof(enc, nbc.IV); err != nil {
			return err
		}
	}
	if nbc.AV != nil {
		if av, err = decodeVoteproof(enc, nbc.AV); err != nil {
			return err
		}
	}

	bc.initVoteproof = iv
	bc.acceptVoteproof = av

	return nil
}
