package block

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
)

type BlockConsensusInfoV0PackJSON struct {
	jsonencoder.HintedHead
	IV base.Voteproof `json:"init_voteproof,omitempty"`
	AV base.Voteproof `json:"accept_voteproof,omitempty"`
}

func (bc BlockConsensusInfoV0) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(BlockConsensusInfoV0PackJSON{
		HintedHead: jsonencoder.NewHintedHead(bc.Hint()),
		IV:         bc.initVoteproof,
		AV:         bc.acceptVoteproof,
	})
}

type BlockConsensusInfoV0UnpackJSON struct {
	IV json.RawMessage `json:"init_voteproof"`
	AV json.RawMessage `json:"accept_voteproof"`
}

func (bc *BlockConsensusInfoV0) UnpackJSON(b []byte, enc *jsonencoder.Encoder) error {
	var nbc BlockConsensusInfoV0UnpackJSON
	if err := enc.Unmarshal(b, &nbc); err != nil {
		return err
	}

	return bc.unpack(enc, nbc.IV, nbc.AV)
}
