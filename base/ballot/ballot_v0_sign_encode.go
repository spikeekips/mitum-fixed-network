package ballot

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

func (sb *SIGNBallotV0) unpack(
	_ encoder.Encoder,
	bb BaseBallotV0,
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

	sb.BaseBallotV0 = bb
	sb.SIGNBallotFactV0 = SIGNBallotFactV0{
		BaseBallotFactV0: bf,
		proposal:         proposal,
		newBlock:         newBlock,
	}

	return nil
}
