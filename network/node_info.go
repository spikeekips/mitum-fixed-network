package network

import (
	"fmt"
	"net/url"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json" // TODO rename to jsonenc, bsonencoder -> bsonenc too
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

var (
	NodeInfoV0Type = hint.MustNewType(0x13, 0x01, "node-info-v0")
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
}

type NodeInfoV0 struct {
	node      base.Node
	networkID base.NetworkID
	state     base.State
	lastBlock block.Manifest
	version   util.Version
	u         string
}

func NewNodeInfoV0(
	node base.Node,
	networkID base.NetworkID,
	state base.State,
	lastBlock block.Manifest,
	version util.Version,
	u string,
) NodeInfoV0 {
	return NodeInfoV0{
		node:      node,
		networkID: networkID,
		state:     state,
		lastBlock: lastBlock,
		version:   version,
		u:         u,
	}
}

func (ni NodeInfoV0) String() string {
	return jsonencoder.ToString(ni)
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

	if err := isvalid.Check([]isvalid.IsValider{ni.lastBlock}, nil, true); err != nil {
		return err
	}

	return nil
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