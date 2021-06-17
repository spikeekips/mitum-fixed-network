package ballot

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

func (ab *ACCEPTV0) unpack(
	enc encoder.Encoder,
	bb BaseBallotV0,
	bf BaseFactV0,
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

	if bVoteproof != nil {
		i, err := base.DecodeVoteproof(bVoteproof, enc)
		if err != nil {
			return err
		}
		ab.voteproof = i
	}

	ab.BaseBallotV0 = bb
	ab.ACCEPTFactV0 = ACCEPTFactV0{
		BaseFactV0: bf,
		proposal:   proposal,
		newBlock:   newBlock,
	}

	return nil
}

func (abf *ACCEPTFactV0) unpack(
	_ encoder.Encoder,
	bf BaseFactV0,
	proposal,
	newBlock valuehash.Hash,
) error {
	if proposal != nil && proposal.Empty() {
		return xerrors.Errorf("empty proposal hash found")
	}

	if newBlock != nil && newBlock.Empty() {
		return xerrors.Errorf("empty newBlock hash found")
	}

	abf.BaseFactV0 = bf
	abf.proposal = proposal
	abf.newBlock = newBlock

	return nil
}
