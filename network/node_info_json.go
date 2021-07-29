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
	UL  string                 `json:"url"`
	PO  map[string]interface{} `json:"policy"`
	SF  []RemoteNode           `json:"suffrage"`
}

func (ni NodeInfoV0) JSONPacker() NodeInfoV0PackerJSON {
	return NodeInfoV0PackerJSON{
		HintedHead: jsonenc.NewHintedHead(ni.Hint()),
		ND:         ni.node,
		NID:        ni.networkID,
		ST:         ni.state,
		LB:         ni.lastBlock,
		VS:         ni.version,
		UL:         ni.u,
		PO:         ni.policy,
		SF:         ni.nodes,
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
	UL  string                 `json:"url"`
	PO  map[string]interface{} `json:"policy"`
	SF  []json.RawMessage      `json:"suffrage"`
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

	return ni.unpack(enc, nni.ND, nni.NID, nni.ST, nni.LB, nni.VS, nni.UL, nni.PO, sf)
}

func (no RemoteNode) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"address":   no.Address,
		"publickey": no.Publickey,
	}

	if len(no.URL) > 0 {
		m["url"] = no.URL
		m["insecure"] = no.Insecure
	}

	return util.JSON.Marshal(m)
}

type RemoteNodeUnpackJSON struct {
	A base.AddressDecoder  `json:"address"`
	P key.PublickeyDecoder `json:"publickey"`
	U string               `json:"url"`
	I bool                 `json:"insecure"`
}

func (no *RemoteNode) unpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uno RemoteNodeUnpackJSON
	if err := util.JSON.Unmarshal(b, &uno); err != nil {
		return err
	}

	return no.unpack(enc, uno.A, uno.P, uno.U, uno.I)
}
