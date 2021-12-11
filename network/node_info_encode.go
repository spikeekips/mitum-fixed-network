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
	co map[string]interface{},
	sf []RemoteNode,
	bci []byte,
) error {
	if err := encoder.Decode(bnode, enc, &ni.node); err != nil {
		return err
	}

	ni.networkID = bnid
	ni.state = st

	var b block.Manifest
	if err := encoder.Decode(blb, enc, &b); err != nil {
		return err
	}

	ni.lastBlock = b
	ni.version = vs
	ni.policy = co
	ni.nodes = sf

	return encoder.Decode(bci, enc, &ni.ci)
}

func (no *RemoteNode) unpack(
	enc encoder.Encoder,
	ba base.AddressDecoder,
	bp key.PublickeyDecoder,
	bci []byte,
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

	return encoder.Decode(bci, enc, &no.ci)
}
