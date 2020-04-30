package state

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

type OperationInfo interface {
	isvalid.IsValider
	hint.Hinter
	util.Byter
	Operation() valuehash.Hash
	Seal() valuehash.Hash
}

type State interface {
	isvalid.IsValider
	hint.Hinter
	valuehash.Hasher
	Key() string
	Value() Value
	GenerateHash() valuehash.Hash
	PreviousBlock() valuehash.Hash
	Operations() []OperationInfo
	Height() base.Height
	CurrentBlock() valuehash.Hash
}

type StateUpdater interface {
	State
	SetValue(Value) error
	SetHash(valuehash.Hash) error
	SetPreviousBlock(valuehash.Hash) error
	AddOperationInfo(OperationInfo) error
	SetCurrentBlock(base.Height, valuehash.Hash) error
}
