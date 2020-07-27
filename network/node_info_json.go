package network

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/policy"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type NodeInfoV0PackerJSON struct {
	jsonenc.HintedHead
	ND  base.Node      `json:"node"`
	NID base.NetworkID `json:"network_id"`
	ST  base.State     `json:"state"`
	LB  block.Manifest `json:"last_block"`
	VS  util.Version   `json:"version"`
	UL  string         `json:"url"`
	PO  policy.Policy  `json:"policy"`
}

func (ni NodeInfoV0) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(NodeInfoV0PackerJSON{
		HintedHead: jsonenc.NewHintedHead(ni.Hint()),
		ND:         ni.node,
		NID:        ni.networkID,
		ST:         ni.state,
		LB:         ni.lastBlock,
		VS:         ni.version,
		UL:         ni.u,
		PO:         ni.policy,
	})
}

type NodeInfoV0UnpackerJSON struct {
	ND  json.RawMessage `json:"node"`
	NID base.NetworkID  `json:"network_id"`
	ST  base.State      `json:"state"`
	LB  json.RawMessage `json:"last_block"`
	VS  util.Version    `json:"version"`
	UL  string          `json:"url"`
	PO  json.RawMessage `json:"policy"`
}

func (ni *NodeInfoV0) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var nni NodeInfoV0UnpackerJSON
	if err := enc.Unmarshal(b, &nni); err != nil {
		return err
	}

	return ni.unpack(enc, nni.ND, nni.NID, nni.ST, nni.LB, nni.VS, nni.UL, nni.PO)
}
