package block

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

func (bc *ConsensusInfoV0) unpack(enc encoder.Encoder, biv, bav, bsi, bpr []byte) error {
	var err error

	var iv, av base.Voteproof
	if biv != nil {
		if iv, err = base.DecodeVoteproof(biv, enc); err != nil {
			return err
		}
	}
	if bav != nil {
		if av, err = base.DecodeVoteproof(bav, enc); err != nil {
			return err
		}
	}

	si, err := DecodeSuffrageInfo(bsi, enc)
	if err != nil {
		return err
	}

	if bpr != nil {
		i, err := ballot.DecodeProposal(bpr, enc)
		if err != nil {
			return err
		}
		bc.proposal = i
	}

	bc.initVoteproof = iv
	bc.acceptVoteproof = av
	bc.suffrageInfo = si

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
