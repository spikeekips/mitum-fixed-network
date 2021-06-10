package ballot

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

func (sb *SIGNV0) unpack(
	_ encoder.Encoder,
	bb BaseBallotV0,
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

	sb.BaseBallotV0 = bb
	sb.SIGNFactV0 = SIGNFactV0{
		BaseFactV0: bf,
		proposal:   proposal,
		newBlock:   newBlock,
	}

	return nil
}
