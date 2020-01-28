package mitum

import (
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/valuehash"
)

type Fact interface {
	isvalid.IsValider
	Hash([]byte) (valuehash.Hash, error)
	Bytes() []byte
}

type FactSeal interface {
	seal.Seal
	Fact() Fact
}
