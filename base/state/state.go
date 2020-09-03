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
	Height() base.Height
	PreviousHeight() base.Height
	Operations() []valuehash.Hash
	GenerateHash() valuehash.Hash
	Merge(State) (State, error)
}

type StateUpdater interface {
	State
	SetHash(valuehash.Hash) error
	SetPreviousHeight(base.Height) error
	SetHeight(base.Height) error
	AddOperation(valuehash.Hash) error
	Reset()
}
