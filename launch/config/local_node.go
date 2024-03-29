package config

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util/encoder"
)

type LocalNode interface {
	Source() map[string]interface{}
	Node
	NetworkID() base.NetworkID
	SetNetworkID(string) error
	Privatekey() key.Privatekey
	SetPrivatekey(string) error
	Network() LocalNetwork
	SetNetwork(LocalNetwork) error
	Storage() Storage
	SetStorage(Storage) error
	Nodes() []RemoteNode
	SetNodes([]RemoteNode) error
	Suffrage() Suffrage
	SetSuffrage(Suffrage) error
	ProposalProcessor() ProposalProcessor
	SetProposalProcessor(ProposalProcessor) error
	Policy() Policy
	SetPolicy(Policy) error
	GenesisOperations() []operation.Operation
	SetGenesisOperations([]operation.Operation) error
	LocalConfig() LocalConfig
	SetLocalConfig(LocalConfig) error
}

type BaseLocalNode struct {
	enc               encoder.Encoder
	source            map[string]interface{}
	address           base.Address
	networkID         base.NetworkID
	privatekey        key.Privatekey
	network           LocalNetwork
	storage           Storage
	nodes             []RemoteNode
	suffrage          Suffrage
	proposalProcessor ProposalProcessor
	policy            Policy
	genesisOperations []operation.Operation
	localConfig       LocalConfig
}

func NewBaseLocalNode(enc encoder.Encoder, source map[string]interface{}) *BaseLocalNode {
	return &BaseLocalNode{
		enc:         enc,
		source:      source,
		network:     EmptyBaseLocalNetwork(),
		storage:     EmptyBaseStorage(),
		policy:      &BasePolicy{},
		localConfig: EmptyDefaultLocalConfig(),
	}
}

func (no BaseLocalNode) Address() base.Address {
	return no.address
}

func (no *BaseLocalNode) SetAddress(s string) error {
	address, err := base.DecodeAddressFromString(s, no.enc)
	if err != nil {
		return errors.Wrapf(err, "invalid address, %q", s)
	}
	no.address = address

	return nil
}

func (no BaseLocalNode) NetworkID() base.NetworkID {
	return no.networkID
}

func (no *BaseLocalNode) SetNetworkID(s string) error {
	no.networkID = base.NetworkID([]byte(s))

	return nil
}

func (no BaseLocalNode) Privatekey() key.Privatekey {
	return no.privatekey
}

func (no *BaseLocalNode) SetPrivatekey(s string) error {
	priv, err := key.DecodePrivatekeyFromString(s, no.enc)
	if err != nil {
		return errors.Wrapf(err, "invalid privatekey, %q", s)
	}
	no.privatekey = priv

	return nil
}

func (no BaseLocalNode) Network() LocalNetwork {
	return no.network
}

func (no *BaseLocalNode) SetNetwork(n LocalNetwork) error {
	no.network = n

	return nil
}

func (no BaseLocalNode) Storage() Storage {
	return no.storage
}

func (no *BaseLocalNode) SetStorage(st Storage) error {
	no.storage = st

	return nil
}

func (no BaseLocalNode) Nodes() []RemoteNode {
	return no.nodes
}

func (no *BaseLocalNode) SetNodes(nodes []RemoteNode) error {
	no.nodes = nodes

	return nil
}

func (no BaseLocalNode) Suffrage() Suffrage {
	return no.suffrage
}

func (no *BaseLocalNode) SetSuffrage(sf Suffrage) error {
	no.suffrage = sf

	return nil
}

func (no BaseLocalNode) ProposalProcessor() ProposalProcessor {
	return no.proposalProcessor
}

func (no *BaseLocalNode) SetProposalProcessor(pp ProposalProcessor) error {
	no.proposalProcessor = pp

	return nil
}

func (no BaseLocalNode) Policy() Policy {
	return no.policy
}

func (no *BaseLocalNode) SetPolicy(p Policy) error {
	no.policy = p

	return nil
}

func (no BaseLocalNode) GenesisOperations() []operation.Operation {
	return no.genesisOperations
}

func (no *BaseLocalNode) SetGenesisOperations(ops []operation.Operation) error {
	no.genesisOperations = ops

	return nil
}

func (no BaseLocalNode) LocalConfig() LocalConfig {
	return no.localConfig
}

func (no *BaseLocalNode) SetLocalConfig(s LocalConfig) error {
	no.localConfig = s

	return nil
}

func (no BaseLocalNode) Source() map[string]interface{} {
	return no.source
}
