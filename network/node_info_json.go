package network

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/util"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
)

type NodeInfoV0PackerJSON struct {
	jsonencoder.HintedHead
	ND  base.Node      `json:"node"`
	NID base.NetworkID `json:"network_id"`
	ST  base.State     `json:"state"`
	LB  block.Manifest `json:"last_block"`
	VS  util.Version   `json:"version"`
	UL  string         `json:"url"`
}

func (ni NodeInfoV0) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(NodeInfoV0PackerJSON{
		HintedHead: jsonencoder.NewHintedHead(ni.Hint()),
		ND:         ni.node,
		NID:        ni.networkID,
		ST:         ni.state,
		LB:         ni.lastBlock,
		VS:         ni.version,
		UL:         ni.u,
	})
}

type NodeInfoV0UnpackerJSON struct {
	ND  json.RawMessage `json:"node"`
	NID base.NetworkID  `json:"network_id"`
	ST  base.State      `json:"state"`
	LB  json.RawMessage `json:"last_block"`
	VS  util.Version    `json:"version"`
	UL  string          `json:"url"`
}

func (ni *NodeInfoV0) UnpackJSON(b []byte, enc *jsonencoder.Encoder) error {
	var nni NodeInfoV0UnpackerJSON
	if err := enc.Unmarshal(b, &nni); err != nil {
		return err
	}

	return ni.unpack(enc, nni.ND, nni.NID, nni.ST, nni.LB, nni.VS, nni.UL)
}
