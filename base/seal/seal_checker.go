package seal

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/isvalid"
)

func IsValidSeal(seal Seal, networkID []byte) error {
	if h, err := seal.GenerateHash(); err != nil {
		return err
	} else if sh := seal.Hash(); !sh.Equal(h) {
		return isvalid.InvalidError.Errorf("hash does not match: seal=%s(%v) generated=%s(%v)", sh, sh.Hint(), h, h.Hint())
	}

	if err := seal.Signer().Verify(
		util.ConcatBytesSlice(seal.BodyHash().Bytes(), networkID),
		seal.Signature(),
	); err != nil {
		return err
	}

	return nil
}
