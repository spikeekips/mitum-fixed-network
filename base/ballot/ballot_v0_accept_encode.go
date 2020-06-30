package ballot

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

func (ab *ACCEPTBallotV0) unpack(
	enc encoder.Encoder,
	bb BaseBallotV0,
	bf BaseBallotFactV0,
	proposal,
	newBlock valuehash.Hash,
	bVoteproof []byte,
) error {
	if proposal != nil && proposal.Empty() {
		return xerrors.Errorf("empty proposal hash found")
	}

	if newBlock != nil && newBlock.Empty() {
		return xerrors.Errorf("empty newBlock hash found")
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
		proposal:         proposal,
		newBlock:         newBlock,
	}
	ab.voteproof = voteproof

	return nil
}

func (abf *ACCEPTBallotFactV0) unpack(
	_ encoder.Encoder,
	bf BaseBallotFactV0,
	proposal,
	newBlock valuehash.Hash,
) error {
	if proposal != nil && proposal.Empty() {
		return xerrors.Errorf("empty proposal hash found")
	}

	if newBlock != nil && newBlock.Empty() {
		return xerrors.Errorf("empty newBlock hash found")
	}

	abf.BaseBallotFactV0 = bf
	abf.proposal = proposal
	abf.newBlock = newBlock

	return nil
}
