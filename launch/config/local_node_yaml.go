package config

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
)

type BaseLocalNodePackerYAML struct {
	Address           base.Address
	NetworkID         string `yaml:"network-id"`
	Privatekey        key.Privatekey
	Network           LocalNetwork `yaml:",omitempty"`
	Storage           Storage
	Nodes             []RemoteNode           `yaml:"nodes,omitempty"`
	Suffrage          map[string]interface{} `yaml:",omitempty"`
	ProposalProcessor map[string]interface{} `yaml:",omitempty"`
	Policy            Policy                 `yaml:",omitempty"`
	GenesisOperations []operation.Operation  `yaml:"genesis-operations,omitempty"`
	TimeServer        string                 `yaml:"timeserver,omitempty"`
}

func NewBaseLocalNodePackerYAMLFromConfig(conf LocalNode) BaseLocalNodePackerYAML {
	return BaseLocalNodePackerYAML{
		Address:           conf.Address(),
		NetworkID:         string(conf.NetworkID()),
		Privatekey:        conf.Privatekey(),
		Network:           conf.Network(),
		Storage:           conf.Storage(),
		Nodes:             conf.Nodes(),
		Policy:            conf.Policy(),
		GenesisOperations: conf.GenesisOperations(),
		TimeServer:        conf.TimeServer(),
	}
}
