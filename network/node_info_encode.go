package network

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

func (ni *NodeInfoV0) unpack(
	enc encoder.Encoder,
	bnode, bnid []byte,
	st base.State,
	blb []byte,
	vs util.Version,
	u string,
	co map[string]interface{},
	sf []RemoteNode,
) error {
	n, err := base.DecodeNode(bnode, enc)
	if err != nil {
		return err
	}
	ni.node = n

	ni.networkID = bnid
	ni.state = st
	b, err := block.DecodeManifest(blb, enc)
	if err != nil {
		return err
	}
	ni.lastBlock = b

	ni.version = vs
	ni.u = u

	ni.policy = co

	ni.nodes = sf

	return nil
}

func (no *RemoteNode) unpack(
	enc encoder.Encoder,
	ba base.AddressDecoder,
	bp key.PublickeyDecoder,
	u string,
	insecure bool,
) error {
	i, err := ba.Encode(enc)
	if err != nil {
		return err
	}
	no.Address = i

	j, err := bp.Encode(enc)
	if err != nil {
		return err
	}
	no.Publickey = j

	if len(u) > 0 {
		no.URL = u
		no.Insecure = insecure
	}

	return nil
}
