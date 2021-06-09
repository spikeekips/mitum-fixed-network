package network

import (
	"net/url"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

var (
	NodeInfoType   = hint.Type("node-info")
	NodeInfoV0Hint = hint.NewHint(NodeInfoType, "v0.0.1")
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
	Policy() map[string]interface{}
	Nodes() []base.Node // Only contains suffrage nodes
}

type NodeInfoV0 struct {
	node      base.Node
	networkID base.NetworkID
	state     base.State
	lastBlock block.Manifest
	version   util.Version
	u         string
	policy    map[string]interface{}
	nodes     []base.Node
}

func NewNodeInfoV0(
	node base.Node,
	networkID base.NetworkID,
	state base.State,
	lastBlock block.Manifest,
	version util.Version,
	u string,
	policy map[string]interface{},
	nodes []base.Node,
	suffrage base.Suffrage,
) NodeInfoV0 {
	// NOTE insert node itself to nodes
	var found bool
	for i := range nodes {
		if nodes[i].Address().Equal(node.Address()) {
			found = true

			break
		}
	}
	if !found {
		nodes = append(nodes, node)
	}

	if suffrage != nil {
		policy["suffrage"] = suffrage.Verbose()
	}

	return NodeInfoV0{
		node:      node,
		networkID: networkID,
		state:     state,
		lastBlock: lastBlock,
		version:   version,
		u:         u,
		policy:    policy,
		nodes:     nodes,
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

func (ni NodeInfoV0) Policy() map[string]interface{} {
	return ni.policy
}

func (ni NodeInfoV0) Nodes() []base.Node {
	return ni.nodes
}
