package seal

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/isvalid"
)

func IsValidSeal(seal Seal, networkID []byte) error {
	if seal.SignedAt().IsZero() {
		return isvalid.InvalidError.Errorf("empty SignedAt")
	}

	if seal.Hash() == nil || seal.BodyHash() == nil {
		return isvalid.InvalidError.Errorf("empty hash")
	}

	if !seal.Hash().Equal(seal.GenerateHash()) {
		return isvalid.InvalidError.Errorf("hash does not match")
	}

	if err := seal.Signer().Verify(
		util.ConcatBytesSlice(seal.BodyHash().Bytes(), networkID),
		seal.Signature(),
	); err != nil {
		return isvalid.InvalidError.Errorf("invalid seal signature: %w", err)
	}

	return nil
}
