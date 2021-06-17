package block

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type ConsensusInfoV0PackJSON struct {
	jsonenc.HintedHead
	IV base.Voteproof  `json:"init_voteproof,omitempty"`
	AV base.Voteproof  `json:"accept_voteproof,omitempty"`
	SI SuffrageInfo    `json:"suffrage_info,omitempty"`
	PR ballot.Proposal `json:"proposal,omitempty"`
}

func (bc ConsensusInfoV0) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(ConsensusInfoV0PackJSON{
		HintedHead: jsonenc.NewHintedHead(bc.Hint()),
		IV:         bc.initVoteproof,
		AV:         bc.acceptVoteproof,
		SI:         bc.suffrageInfo,
		PR:         bc.proposal,
	})
}

type ConsensusInfoV0UnpackJSON struct {
	IV json.RawMessage `json:"init_voteproof"`
	AV json.RawMessage `json:"accept_voteproof"`
	SI json.RawMessage `json:"suffrage_info"`
	PR json.RawMessage `json:"proposal"`
}

func (bc *ConsensusInfoV0) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var nbc ConsensusInfoV0UnpackJSON
	if err := enc.Unmarshal(b, &nbc); err != nil {
		return err
	}

	return bc.unpack(enc, nbc.IV, nbc.AV, nbc.SI, nbc.PR)
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
	PR base.AddressDecoder `json:"proposer"`
	NS json.RawMessage     `json:"nodes"`
}

func (si *SuffrageInfoV0) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var nsi SuffrageInfoV0UnpackJSON
	if err := enc.Unmarshal(b, &nsi); err != nil {
		return err
	}

	return si.unpack(enc, nsi.PR, nsi.NS)
}
