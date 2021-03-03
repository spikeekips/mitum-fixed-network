package config

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type BaseLocalNodePackerJSON struct {
	Address           base.Address           `json:"address"`
	NetworkID         string                 `json:"network_id"`
	Privatekey        key.Privatekey         `json:"privatekey"`
	Network           LocalNetwork           `json:"network,omitempty"`
	Storage           Storage                `json:"storage"`
	Nodes             []RemoteNode           `json:"nodes,omitempty"`
	Suffrage          map[string]interface{} `json:"suffrage,omitempty"`
	ProposalProcessor map[string]interface{} `json:"proposal_processor,omitempty"`
	Policy            Policy                 `json:"policy,omitempty"`
	GenesisOperations []operation.Operation  `json:"genesis_operations,omitempty"`
	TimeServer        string                 `json:"timeserver,omitempty"`
}

func (no BaseLocalNode) MarshalJSON() ([]byte, error) {
	var suffrage map[string]interface{}
	if i, ok := no.Source()["suffrage"].(map[string]interface{}); ok {
		suffrage = i
	}

	var proposalProcessor map[string]interface{}
	if i, ok := no.Source()["proposal-processor"].(map[string]interface{}); ok {
		proposalProcessor = i
	}

	return jsonenc.Marshal(BaseLocalNodePackerJSON{
		Address:           no.Address(),
		NetworkID:         string(no.NetworkID()),
		Privatekey:        no.Privatekey(),
		Network:           no.Network(),
		Storage:           no.Storage(),
		Nodes:             no.Nodes(),
		Suffrage:          suffrage,
		ProposalProcessor: proposalProcessor,
		Policy:            no.Policy(),
		GenesisOperations: no.GenesisOperations(),
		TimeServer:        no.TimeServer(),
	})
}
