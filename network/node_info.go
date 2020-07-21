package network

import (
	"fmt"
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
	NodeInfoV0Type = hint.MustNewType(0x01, 0x86, "node-info-v0")
	NodeInfoV0Hint = hint.MustHint(NodeInfoV0Type, "0.0.1")
)

type NodeInfo interface {
	fmt.Stringer
	isvalid.IsValider
	hint.Hinter
	base.Node
	NetworkID() base.NetworkID
	State() base.State
	LastBlock() block.Manifest
	Version() util.Version
	URL() string
	Policy() base.PolicyOperationBody
}

type NodeInfoV0 struct {
	node      base.Node
	networkID base.NetworkID
	state     base.State
	lastBlock block.Manifest
	version   util.Version
	u         string
	policy    base.PolicyOperationBody
}

func NewNodeInfoV0(
	node base.Node,
	networkID base.NetworkID,
	state base.State,
	lastBlock block.Manifest,
	version util.Version,
	u string,
	policy base.PolicyOperationBody,
) NodeInfoV0 {
	return NodeInfoV0{
		node:      node,
		networkID: networkID,
		state:     state,
		lastBlock: lastBlock,
		version:   version,
		u:         u,
		policy:    policy,
	}
}

func (ni NodeInfoV0) String() string {
	return jsonenc.ToString(ni)
}

func (ni NodeInfoV0) Bytes() []byte {
	return util.ConcatBytesSlice(
		ni.node.Bytes(),
		ni.networkID.Bytes(),
		ni.state.Bytes(),
		ni.lastBlock.Hash().Bytes(),
		ni.version.Bytes(),
		[]byte(ni.u),
		ni.policy.Bytes(),
	)
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

func (ni NodeInfoV0) Policy() base.PolicyOperationBody {
	return ni.policy
}
