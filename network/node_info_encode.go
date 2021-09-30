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

	ni.policy = co

	ni.nodes = sf

	uci, err := DecodeConnInfo(bci, enc)
	if err != nil {
		return err
	}
	ni.ci = uci

	return nil
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

	if len(bci) > 0 {
		i, err := DecodeConnInfo(bci, enc)
		if err != nil {
			return err
		}

		no.ci = i
	}

	return nil
}
