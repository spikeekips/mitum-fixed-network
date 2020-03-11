package state

import (
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

type OperationInfo interface {
	isvalid.IsValider
	hint.Hinter
	Operation() valuehash.Hash
	Seal() valuehash.Hash
	Bytes() []byte
}

// TODO the target of value should provide Bytes() and NewXXFromBytes()
type State interface {
	isvalid.IsValider
	hint.Hinter
	Hash() valuehash.Hash
	Key() string
	Value() util.Byter
	ValueHash() valuehash.Hash
	GenerateHash() valuehash.Hash
	PreviousBlock() valuehash.Hash
	Operations() []OperationInfo
	CurrentBlock() valuehash.Hash
}

type StateUpdater interface {
	State
	SetValue(util.Byter, valuehash.Hash) error
	SetHash(valuehash.Hash) error
	SetPreviousBlock(valuehash.Hash) error
	AddOperationInfo(OperationInfo) error
	SetCurrentBlock(valuehash.Hash) error
}
