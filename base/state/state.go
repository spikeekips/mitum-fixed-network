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
	SetValue(Value) (State, error)
	GenerateHash() valuehash.Hash
	PreviousBlock() valuehash.Hash
	Operations() []valuehash.Hash
	Height() base.Height
	CurrentBlock() valuehash.Hash
	Merge(State) (State, error)
}

type StateUpdater interface {
	State
	SetHash(valuehash.Hash) error
	SetPreviousBlock(valuehash.Hash) error
	AddOperation(valuehash.Hash) error
	SetCurrentBlock(base.Height, valuehash.Hash) error
	Reset()
}
