package mitum

import (
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/valuehash"
)

type Ballot interface {
	isvalid.IsValider
	SealBody
	Height() Height
	Round() Round
	Stage() Stage
	Node() Address
	BaseBlock() valuehash.Hash
}
