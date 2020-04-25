package ballot

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/encoder"
)

func (ab *ACCEPTBallotV0) unpack(
	enc encoder.Encoder,
	bb BaseBallotV0,
	bf BaseBallotFactV0,
	bProposal,
	bNewBlock,
	bVoteproof []byte,
) error {
	var epr, enb valuehash.Hash
	if i, err := valuehash.Decode(enc, bProposal); err != nil {
		return err
	} else {
		epr = i
	}

	if i, err := valuehash.Decode(enc, bNewBlock); err != nil {
		return err
	} else {
		enb = i
	}

	var voteproof base.Voteproof
	if bVoteproof != nil {
		if i, err := base.DecodeVoteproof(enc, bVoteproof); err != nil {
			return err
		} else {
			voteproof = i
		}
	}

	ab.BaseBallotV0 = bb
	ab.ACCEPTBallotFactV0 = ACCEPTBallotFactV0{
		BaseBallotFactV0: bf,
		proposal:         epr,
		newBlock:         enb,
	}
	ab.voteproof = voteproof

	return nil
}

func (abf *ACCEPTBallotFactV0) unpack(enc encoder.Encoder, bf BaseBallotFactV0, bProposal, bNewBlock []byte) error {
	var err error

	var pr, nb valuehash.Hash
	if pr, err = valuehash.Decode(enc, bProposal); err != nil {
		return err
	}
	if nb, err = valuehash.Decode(enc, bNewBlock); err != nil {
		return err
	}

	abf.BaseBallotFactV0 = bf
	abf.proposal = pr
	abf.newBlock = nb

	return nil
}
