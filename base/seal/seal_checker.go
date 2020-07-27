package seal

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/isvalid"
)

func IsValidSeal(seal Seal, networkID []byte) error {
	if !seal.Hash().Equal(seal.GenerateHash()) {
		return isvalid.InvalidError.Errorf("hash does not match")
	}

	if err := seal.Signer().Verify(
		util.ConcatBytesSlice(seal.BodyHash().Bytes(), networkID),
		seal.Signature(),
	); err != nil {
		return err
	}

	return nil
}
