package network

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type NodeInfoV0PackerJSON struct {
	jsonenc.HintedHead
	ND  base.Node              `json:"node"`
	NID base.NetworkID         `json:"network_id"`
	ST  base.State             `json:"state"`
	LB  block.Manifest         `json:"last_block"`
	VS  util.Version           `json:"version"`
	PO  map[string]interface{} `json:"policy"`
	SF  []RemoteNode           `json:"suffrage"`
	CI  ConnInfo               `json:"conninfo"`
}

func (ni NodeInfoV0) JSONPacker() NodeInfoV0PackerJSON {
	return NodeInfoV0PackerJSON{
		HintedHead: jsonenc.NewHintedHead(ni.Hint()),
		ND:         ni.node,
		NID:        ni.networkID,
		ST:         ni.state,
		LB:         ni.lastBlock,
		VS:         ni.version,
		PO:         ni.policy,
		SF:         ni.nodes,
		CI:         ni.ci,
	}
}

func (ni NodeInfoV0) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(ni.JSONPacker())
}

type NodeInfoV0UnpackerJSON struct {
	ND  json.RawMessage        `json:"node"`
	NID base.NetworkID         `json:"network_id"`
	ST  base.State             `json:"state"`
	LB  json.RawMessage        `json:"last_block"`
	VS  util.Version           `json:"version"`
	PO  map[string]interface{} `json:"policy"`
	SF  []json.RawMessage      `json:"suffrage"`
	CI  json.RawMessage        `json:"conninfo"`
}

func (ni *NodeInfoV0) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var nni NodeInfoV0UnpackerJSON
	if err := enc.Unmarshal(b, &nni); err != nil {
		return err
	}

	sf := make([]RemoteNode, len(nni.SF))
	for i := range nni.SF {
		var r RemoteNode
		if err := r.unpackJSON(nni.SF[i], enc); err != nil {
			return err
		}

		sf[i] = r
	}

	return ni.unpack(enc, nni.ND, nni.NID, nni.ST, nni.LB, nni.VS, nni.PO, sf, nni.CI)
}

func (no RemoteNode) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(map[string]interface{}{
		"address":   no.Address,
		"publickey": no.Publickey,
		"conninfo":  no.ci,
	})
}

type RemoteNodeUnpackJSON struct {
	A  base.AddressDecoder  `json:"address"`
	P  key.PublickeyDecoder `json:"publickey"`
	CI json.RawMessage      `json:"conninfo"`
}

func (no *RemoteNode) unpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uno RemoteNodeUnpackJSON
	if err := util.JSON.Unmarshal(b, &uno); err != nil {
		return err
	}

	return no.unpack(enc, uno.A, uno.P, uno.CI)
}
