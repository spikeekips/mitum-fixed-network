package ballot

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

func (ib *INITBallotV0) unpack(
	enc encoder.Encoder,
	bb BaseBallotV0,
	bf BaseBallotFactV0,
	previousBlock valuehash.Hash,
	bVoteproof,
	bAVoteproof []byte,
) error {
	if previousBlock != nil && previousBlock.Empty() {
		return xerrors.Errorf("empty previous_block hash found")
	}

	var voteproof, acceptVoteproof base.Voteproof
	if bVoteproof != nil {
		if i, err := base.DecodeVoteproof(enc, bVoteproof); err != nil {
			return err
		} else {
			voteproof = i
		}
	}

	if bAVoteproof != nil {
		if i, err := base.DecodeVoteproof(enc, bAVoteproof); err != nil {
			return err
		} else {
			acceptVoteproof = i
		}
	}

	ib.BaseBallotV0 = bb
	ib.INITBallotFactV0 = INITBallotFactV0{
		BaseBallotFactV0: bf,
		previousBlock:    previousBlock,
	}
	ib.voteproof = voteproof
	ib.acceptVoteproof = acceptVoteproof

	return nil
}

func (ibf *INITBallotFactV0) unpack(
	_ encoder.Encoder,
	bf BaseBallotFactV0,
	previousBlock valuehash.Hash,
) error {
	if previousBlock != nil && previousBlock.Empty() {
		return xerrors.Errorf("empty previous_block hash found")
	}

	ibf.BaseBallotFactV0 = bf
	ibf.previousBlock = previousBlock

	return nil
}
