package network

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
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
	bsf [][]byte,
) error {
	n, err := base.DecodeNode(enc, bnode)
	if err != nil {
		return err
	}
	ni.node = n

	ni.networkID = bnid
	ni.state = st
	b, err := block.DecodeManifest(enc, blb)
	if err != nil {
		return err
	}
	ni.lastBlock = b

	ni.version = vs
	ni.u = u

	ni.policy = co

	sf := make([]base.Node, len(bsf))
	for i := range bsf {
		n, err := base.DecodeNode(enc, bsf[i])
		if err != nil {
			return err
		}
		sf[i] = n
	}

	ni.nodes = sf

	return nil
}
