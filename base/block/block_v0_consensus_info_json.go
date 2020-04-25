package block

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

type BlockConsensusInfoV0PackJSON struct {
	encoder.JSONPackHintedHead
	IV base.Voteproof `json:"init_voteproof,omitempty"`
	AV base.Voteproof `json:"accept_voteproof,omitempty"`
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

	return bc.unpack(enc, nbc.IV, nbc.AV)
}
