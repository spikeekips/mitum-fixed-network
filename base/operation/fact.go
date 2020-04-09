package operation

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/isvalid"
)

type EmbededFact interface {
	Fact() base.Fact
	FactHash() valuehash.Hash
	FactSignature() key.Signature
}

type FactSeal interface {
	seal.Seal
	EmbededFact
}

func IsValidEmbededFact(signer key.Publickey, ef EmbededFact, b []byte) error {
	if ef.Fact() == nil {
		return isvalid.InvalidError.Errorf("EmbdedFact has empty Fact()")
	}
	if ef.FactHash() == nil {
		return isvalid.InvalidError.Errorf("EmbdedFact has empty FactHash()")
	}
	if ef.FactSignature() == nil {
		return isvalid.InvalidError.Errorf("EmbdedFact has empty FactSignature()")
	}

	if err := signer.Verify(util.ConcatBytesSlice(ef.FactHash().Bytes(), b), ef.FactSignature()); err != nil {
		return err
	}

	return signer.Verify(
		util.ConcatBytesSlice(ef.Fact().Hash().Bytes(), b),
		ef.FactSignature(),
	)
}
