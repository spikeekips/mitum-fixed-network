package block

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/util/encoder"
)

func (bc *ConsensusInfoV0) unpack(enc encoder.Encoder, biv, bav, bsi, bpr []byte) error {
	var err error

	var iv, av base.Voteproof
	if biv != nil {
		if iv, err = base.DecodeVoteproof(enc, biv); err != nil {
			return err
		}
	}
	if bav != nil {
		if av, err = base.DecodeVoteproof(enc, bav); err != nil {
			return err
		}
	}

	si, err := DecodeSuffrageInfo(enc, bsi)
	if err != nil {
		return err
	}

	if bpr != nil {
		i, err := ballot.DecodeProposal(enc, bpr)
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

func (si *SuffrageInfoV0) unpack(enc encoder.Encoder, bpr base.AddressDecoder, bns [][]byte) error {
	i, err := bpr.Encode(enc)
	if err != nil {
		return err
	}
	si.proposer = i

	si.nodes = make([]base.Node, len(bns))
	for i := range bns {
		n, err := base.DecodeNode(enc, bns[i])
		if err != nil {
			return err
		}
		si.nodes[i] = n
	}

	return nil
}
