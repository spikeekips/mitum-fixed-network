package operation

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

type Fact interface {
	isvalid.IsValider
	hint.Hinter
	Hash() valuehash.Hash
	Bytes() []byte
}

type EmbededFact interface {
	Fact() Fact
	FactHash() valuehash.Hash
	FactSignature() key.Signature
}

type FactSeal interface {
	seal.Seal
	EmbededFact
}

func IsValidEmbededFact(signer key.Publickey, ef EmbededFact, b []byte) error {
	if ef.Fact() == nil {
		return xerrors.Errorf("EmbdedFact has empty Fact()")
	}
	if ef.FactHash() == nil {
		return xerrors.Errorf("EmbdedFact has empty FactHash()")
	}
	if ef.FactSignature() == nil {
		return xerrors.Errorf("EmbdedFact has empty FactSignature()")
	}

	return signer.Verify(
		util.ConcatSlice([][]byte{ef.FactHash().Bytes(), b}),
		ef.FactSignature(),
	)
}
