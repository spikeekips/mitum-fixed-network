package network

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

var (
	NodeInfoType     = hint.Type("node-info")
	NodeInfoV0Hint   = hint.NewHint(NodeInfoType, "v0.0.1")
	NodeInfoV0Hinter = NodeInfoV0{BaseHinter: hint.NewBaseHinter(NodeInfoV0Hint)}
)

type NodeInfo interface {
	isvalid.IsValider
	hint.Hinter
	base.Node
	NetworkID() base.NetworkID
	State() base.State
	LastBlock() block.Manifest
	Version() util.Version
	ConnInfo() ConnInfo
	Policy() map[string]interface{}
	Nodes() []RemoteNode // Only contains suffrage nodes
}

type NodeInfoV0 struct {
	hint.BaseHinter
	node      base.Node
	networkID base.NetworkID
	state     base.State
	lastBlock block.Manifest
	version   util.Version
	policy    map[string]interface{}
	nodes     []RemoteNode
	ci        ConnInfo
}

func NewNodeInfoV0(
	node base.Node,
	networkID base.NetworkID,
	state base.State,
	lastBlock block.Manifest,
	version util.Version,
	policy map[string]interface{},
	nodes []RemoteNode,
	suffrage base.Suffrage,
	ci ConnInfo,
) NodeInfoV0 {
	if suffrage != nil {
		policy["suffrage"] = suffrage.Verbose()
	}

	return NodeInfoV0{
		BaseHinter: hint.NewBaseHinter(NodeInfoV0Hint),
		node:       node,
		networkID:  networkID,
		state:      state,
		lastBlock:  lastBlock,
		version:    version,
		policy:     policy,
		nodes:      nodes,
		ci:         ci,
	}
}

func (ni NodeInfoV0) String() string {
	return jsonenc.ToString(ni)
}

func (NodeInfoV0) Bytes() []byte {
	return nil
}

func (ni NodeInfoV0) IsValid([]byte) error {
	if err := ni.node.IsValid(nil); err != nil {
		return err
	}

	if err := ni.networkID.IsValid(nil); err != nil {
		return err
	}

	if err := isvalid.Check(nil, false, ni.state, ni.version, ni.ci); err != nil {
		return err
	}

	return isvalid.Check(nil, true, ni.lastBlock)
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

func (ni NodeInfoV0) ConnInfo() ConnInfo {
	return ni.ci
}

func (ni NodeInfoV0) Policy() map[string]interface{} {
	return ni.policy
}

func (ni NodeInfoV0) Nodes() []RemoteNode {
	return ni.nodes
}

type RemoteNode struct {
	Address   base.Address
	Publickey key.Publickey
	ci        ConnInfo
}

func NewRemoteNode(no base.Node, connInfo ConnInfo) RemoteNode {
	return RemoteNode{Address: no.Address(), Publickey: no.Publickey(), ci: connInfo}
}

func (no RemoteNode) ConnInfo() ConnInfo {
	return no.ci
}

func NewRemoteNodeFromNodeInfo(ni NodeInfo) RemoteNode {
	return RemoteNode{Address: ni.Address(), Publickey: ni.Publickey(), ci: ni.ConnInfo()}
}
