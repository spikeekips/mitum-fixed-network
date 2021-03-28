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
	if n, err := base.DecodeNode(enc, bnode); err != nil {
		return err
	} else {
		ni.node = n
	}

	ni.networkID = bnid
	ni.state = st
	if b, err := block.DecodeManifest(enc, blb); err != nil {
		return err
	} else {
		ni.lastBlock = b
	}

	ni.version = vs
	ni.u = u

	ni.policy = co

	sf := make([]base.Node, len(bsf))
	for i := range bsf {
		if n, err := base.DecodeNode(enc, bsf[i]); err != nil {
			return err
		} else {
			sf[i] = n
		}
	}

	ni.nodes = sf

	return nil
}
