package block

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
)

func (bc *BlockConsensusInfoV0) unpack(enc encoder.Encoder, biv, bav, bsi []byte) error {
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

	var si SuffrageInfo
	if v, err := decodeSuffrageInfo(enc, bsi); err != nil {
		return err
	} else {
		si = v
	}

	bc.initVoteproof = iv
	bc.acceptVoteproof = av
	bc.suffrageInfo = si

	return nil
}

func (si *SuffrageInfoV0) unpack(enc encoder.Encoder, bpr base.AddressDecoder, bns [][]byte) error {
	var proposer base.Address
	if pr, err := bpr.Encode(enc); err != nil {
		return err
	} else {
		proposer = pr
	}

	var ns []base.Node
	for _, b := range bns {
		if n, err := base.DecodeNode(enc, b); err != nil {
			return err
		} else {
			ns = append(ns, n)
		}
	}

	si.proposer = proposer
	si.nodes = ns

	return nil
}
