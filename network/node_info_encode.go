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
	bsf []byte,
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

	hsf, err := enc.DecodeSlice(bsf)
	if err != nil {
		return err
	}

	sf := make([]base.Node, len(hsf))
	for i := range hsf {
		j, ok := hsf[i].(base.Node)
		if !ok {
			return util.WrongTypeError.Errorf("expected base.Node, not %T", hsf[i])
		}

		sf[i] = j
	}

	ni.nodes = sf

	return nil
}
