package isaac

import (
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/valuehash"
)

type Fact interface {
	isvalid.IsValider
	hint.Hinter
	Hash() valuehash.Hash
	Bytes() []byte
}

type FactSeal interface {
	seal.Seal
	Fact() Fact
	FactHash() valuehash.Hash
	FactSignature() key.Signature
}
