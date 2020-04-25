package block

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
)

func (bc *BlockConsensusInfoV0) unpack(enc encoder.Encoder, biv, bav []byte) error {
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

	bc.initVoteproof = iv
	bc.acceptVoteproof = av

	return nil
}
