package network

import (
	"net/url"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/policy"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

var (
	NodeInfoV0Type = hint.MustNewType(0x01, 0x86, "node-info-v0")
	NodeInfoV0Hint = hint.MustHint(NodeInfoV0Type, "0.0.1")
)

type NodeInfo interface {
	isvalid.IsValider
	hint.Hinter
	base.Node
	NetworkID() base.NetworkID
	State() base.State
	LastBlock() block.Manifest
	Version() util.Version
	URL() string
	Policy() policy.Policy
	Config() map[string]interface{}
	Suffrage() []base.Node
}

type NodeInfoV0 struct {
	node      base.Node
	networkID base.NetworkID
	state     base.State
	lastBlock block.Manifest
	version   util.Version
	u         string
	policy    policy.Policy
	config    map[string]interface{}
	suffrage  []base.Node
}

func NewNodeInfoV0(
	node base.Node,
	networkID base.NetworkID,
	state base.State,
	lastBlock block.Manifest,
	version util.Version,
	u string,
	policy policy.Policy,
	config map[string]interface{},
	suffrage []base.Node,
) NodeInfoV0 {
	// NOTE insert node itself to suffrage
	var found bool
	for i := range suffrage {
		if suffrage[i].Address().Equal(node.Address()) {
			found = true

			break
		}
	}
	if !found {
		suffrage = append(suffrage, node)
	}

	return NodeInfoV0{
		node:      node,
		networkID: networkID,
		state:     state,
		lastBlock: lastBlock,
		version:   version,
		u:         u,
		policy:    policy,
		config:    config,
		suffrage:  suffrage,
	}
}

func (ni NodeInfoV0) String() string {
	return jsonenc.ToString(ni)
}

func (ni NodeInfoV0) Bytes() []byte {
	return nil
}

func (ni NodeInfoV0) Hint() hint.Hint {
	return NodeInfoV0Hint
}

func (ni NodeInfoV0) IsValid([]byte) error {
	if err := ni.networkID.IsValid(nil); err != nil {
		return err
	}

	if _, err := url.Parse(ni.u); err != nil {
		return isvalid.InvalidError.Wrap(err).Errorf("invalid node info url")
	}

	if err := isvalid.Check([]isvalid.IsValider{ni.state, ni.version}, nil, false); err != nil {
		return err
	}

	return isvalid.Check([]isvalid.IsValider{ni.lastBlock}, nil, true)
}

func (ni NodeInfoV0) Address() base.Address {
	return ni.node.Address()
}

func (ni NodeInfoV0) Publickey() key.Publickey {
	return ni.node.Publickey()
}

func (ni NodeInfoV0) NetworkID() base.NetworkID {
	return ni.networkID
}

func (ni NodeInfoV0) State() base.State {
	return ni.state
}

func (ni NodeInfoV0) LastBlock() block.Manifest {
	return ni.lastBlock
}

func (ni NodeInfoV0) Version() util.Version {
	return ni.version
}

func (ni NodeInfoV0) URL() string {
	return ni.u
}

func (ni NodeInfoV0) Policy() policy.Policy {
	return ni.policy
}

func (ni NodeInfoV0) Config() map[string]interface{} {
	return ni.config
}

func (ni NodeInfoV0) Suffrage() []base.Node {
	return ni.suffrage
}
