package block

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/isvalid"
)

func (bc *ConsensusInfoV0) unpack(enc encoder.Encoder, biv, bav, bsi, bpr []byte) error {
	var err error

	if err = encoder.Decode(biv, enc, &bc.initVoteproof); err != nil {
		return err
	}

	if err = encoder.Decode(bav, enc, &bc.acceptVoteproof); err != nil {
		return err
	}

	if err := encoder.Decode(bsi, enc, &bc.suffrageInfo); err != nil {
		return err
	}

	if err := encoder.Decode(bpr, enc, &bc.sfs); err != nil {
		return isvalid.InvalidError.Errorf("failed to decode consensus info: %w", err)
	}

	return nil
}

func (si *SuffrageInfoV0) unpack(enc encoder.Encoder, bpr base.AddressDecoder, bns []byte) error {
	i, err := bpr.Encode(enc)
	if err != nil {
		return err
	}
	si.proposer = i

	hinters, err := enc.DecodeSlice(bns)
	if err != nil {
		return err
	}

	si.nodes = make([]base.Node, len(hinters))
	for i := range hinters {
		j, ok := hinters[i].(base.Node)
		if !ok {
			return util.WrongTypeError.Errorf("expected base.Node, not %T", hinters[i])
		}
		si.nodes[i] = j
	}

	return nil
}
