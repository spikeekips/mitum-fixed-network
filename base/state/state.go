package state

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

type State interface {
	isvalid.IsValider
	hint.Hinter
	valuehash.Hasher
	Key() string
	Value() Value
	GenerateHash() valuehash.Hash
	PreviousBlock() valuehash.Hash
	Operations() []valuehash.Hash
	Height() base.Height
	CurrentBlock() valuehash.Hash
}

type StateUpdater interface {
	State
	SetValue(Value) error
	SetHash(valuehash.Hash) error
	SetPreviousBlock(valuehash.Hash) error
	AddOperation(valuehash.Hash) error
	SetCurrentBlock(base.Height, valuehash.Hash) error
}
