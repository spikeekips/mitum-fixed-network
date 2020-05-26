package ballot

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/encoder"
)

func (ib *INITBallotV0) unpack(
	enc encoder.Encoder,
	bb BaseBallotV0,
	bf BaseBallotFactV0,
	bPreviousBlock []byte,
	bVoteproof []byte,
) error {
	var epb valuehash.Hash
	if i, err := valuehash.Decode(enc, bPreviousBlock); err != nil {
		return err
	} else {
		epb = i
	}

	var voteproof base.Voteproof
	if bVoteproof != nil {
		if i, err := base.DecodeVoteproof(enc, bVoteproof); err != nil {
			return err
		} else {
			voteproof = i
		}
	}

	ib.BaseBallotV0 = bb
	ib.INITBallotFactV0 = INITBallotFactV0{
		BaseBallotFactV0: bf,
		previousBlock:    epb,
	}
	ib.voteproof = voteproof

	return nil
}

func (ibf *INITBallotFactV0) unpack(
	enc encoder.Encoder,
	bf BaseBallotFactV0,
	bPreviousBlock []byte,
) error {
	var err error

	var pb valuehash.Hash
	if pb, err = valuehash.Decode(enc, bPreviousBlock); err != nil {
		return err
	}

	ibf.BaseBallotFactV0 = bf
	ibf.previousBlock = pb

	return nil
}
