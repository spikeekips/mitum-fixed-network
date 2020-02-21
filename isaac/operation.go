package isaac

import (
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/valuehash"
)

var (
	OperationType hint.Type = hint.Type{0x06, 0x00}
)

type Operation interface {
	isvalid.IsValider
	hint.Hinter
	Bytes() []byte
	Hash() valuehash.Hash
	Signer() key.Publickey
	Signature() key.Signature
	Fact() Fact
}
