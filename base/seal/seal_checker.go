package seal

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/isvalid"
)

func IsValidSeal(seal Seal, b []byte) error {
	if h, err := seal.GenerateHash(); err != nil {
		return err
	} else if !seal.Hash().Equal(h) {
		return isvalid.InvalidError.Errorf("hash does not match")
	}

	if err := seal.Signer().Verify(
		util.ConcatBytesSlice(seal.BodyHash().Bytes(), b),
		seal.Signature(),
	); err != nil {
		return err
	}

	return nil
}
