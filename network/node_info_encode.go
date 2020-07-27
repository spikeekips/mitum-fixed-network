package network

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/policy"
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
	bpo []byte,
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

	if len(bpo) > 0 {
		if p, err := policy.DecodePolicyV0(enc, bpo); err != nil {
			return err
		} else {
			ni.policy = p
		}
	}

	return nil
}
