package state

import (
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/valuehash"
)

type OperationInfo interface {
	isvalid.IsValider
	hint.Hinter
	Operation() valuehash.Hash
	Seal() valuehash.Hash
	Bytes() []byte
}

type State interface {
	isvalid.IsValider
	hint.Hinter
	Hash() valuehash.Hash
	Key() string
	Value() interface{}
	ValueHash() valuehash.Hash
	GenerateHash() valuehash.Hash
	PreviousBlock() valuehash.Hash
	Operations() []OperationInfo
}

type StateUpdater interface {
	State
	SetValue(interface{}, valuehash.Hash) error
	SetHash(valuehash.Hash) error
	SetPreviousBlock(valuehash.Hash) error
	AddOperationInfo(OperationInfo) error
}
