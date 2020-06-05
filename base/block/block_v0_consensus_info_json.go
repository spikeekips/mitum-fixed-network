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
	SI SuffrageInfo   `json:"suffrage_info,omitempty"`
}

func (bc BlockConsensusInfoV0) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(BlockConsensusInfoV0PackJSON{
		HintedHead: jsonencoder.NewHintedHead(bc.Hint()),
		IV:         bc.initVoteproof,
		AV:         bc.acceptVoteproof,
		SI:         bc.suffrageInfo,
	})
}

type BlockConsensusInfoV0UnpackJSON struct {
	IV json.RawMessage `json:"init_voteproof"`
	AV json.RawMessage `json:"accept_voteproof"`
	SI json.RawMessage `json:"suffrage_info"`
}

func (bc *BlockConsensusInfoV0) UnpackJSON(b []byte, enc *jsonencoder.Encoder) error {
	var nbc BlockConsensusInfoV0UnpackJSON
	if err := enc.Unmarshal(b, &nbc); err != nil {
		return err
	}

	return bc.unpack(enc, nbc.IV, nbc.AV, nbc.SI)
}

type SuffrageInfoV0PackJSON struct {
	jsonencoder.HintedHead
	PR base.Address `json:"proposer"`
	NS []base.Node  `json:"nodes"`
}

func (si SuffrageInfoV0) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(SuffrageInfoV0PackJSON{
		HintedHead: jsonencoder.NewHintedHead(si.Hint()),
		PR:         si.proposer,
		NS:         si.nodes,
	})
}

type SuffrageInfoV0UnpackJSON struct {
	PR json.RawMessage   `json:"proposer"`
	NS []json.RawMessage `json:"nodes"`
}

func (si *SuffrageInfoV0) UnpackJSON(b []byte, enc *jsonencoder.Encoder) error {
	var nsi SuffrageInfoV0UnpackJSON
	if err := enc.Unmarshal(b, &nsi); err != nil {
		return err
	}

	var bsn [][]byte
	for _, n := range nsi.NS {
		bsn = append(bsn, n)
	}

	return si.unpack(enc, nsi.PR, bsn)
}
