package block

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type BlockConsensusInfoV0PackJSON struct {
	jsonenc.HintedHead
	IV base.Voteproof `json:"init_voteproof,omitempty"`
	AV base.Voteproof `json:"accept_voteproof,omitempty"`
	SI SuffrageInfo   `json:"suffrage_info,omitempty"`
}

func (bc BlockConsensusInfoV0) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(BlockConsensusInfoV0PackJSON{
		HintedHead: jsonenc.NewHintedHead(bc.Hint()),
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

func (bc *BlockConsensusInfoV0) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var nbc BlockConsensusInfoV0UnpackJSON
	if err := enc.Unmarshal(b, &nbc); err != nil {
		return err
	}

	return bc.unpack(enc, nbc.IV, nbc.AV, nbc.SI)
}

type SuffrageInfoV0PackJSON struct {
	jsonenc.HintedHead
	PR base.Address `json:"proposer"`
	NS []base.Node  `json:"nodes"`
}

func (si SuffrageInfoV0) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(SuffrageInfoV0PackJSON{
		HintedHead: jsonenc.NewHintedHead(si.Hint()),
		PR:         si.proposer,
		NS:         si.nodes,
	})
}

type SuffrageInfoV0UnpackJSON struct {
	PR json.RawMessage   `json:"proposer"`
	NS []json.RawMessage `json:"nodes"`
}

func (si *SuffrageInfoV0) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var nsi SuffrageInfoV0UnpackJSON
	if err := enc.Unmarshal(b, &nsi); err != nil {
		return err
	}

	bsn := make([][]byte, len(nsi.NS))
	for i := range nsi.NS {
		bsn[i] = nsi.NS[i]
	}

	return si.unpack(enc, nsi.PR, bsn)
}
