package state

import (
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
	"golang.org/x/xerrors"
)

var (
	StateV0Hint         hint.Hint = hint.MustHint(hint.Type{0x11, 0x00}, "0.0.1")
	OperationInfoV0Hint hint.Hint = hint.MustHint(hint.Type{0x11, 0x01}, "0.0.1")
)

type StateV0 struct {
	h             valuehash.Hash
	key           string
	value         interface{}
	valueHash     valuehash.Hash
	previousBlock valuehash.Hash
	operations    []OperationInfo
	currentBlock  valuehash.Hash
}

func NewStateV0(
	key string,
	value interface{},
	valueHash valuehash.Hash,
	previousBlock valuehash.Hash,
) (*StateV0, error) {
	st := &StateV0{
		key:           key,
		value:         value,
		valueHash:     valueHash,
		previousBlock: previousBlock,
	}

	return st, nil
}

func (st StateV0) IsValid([]byte) error {
	if len(st.key) < 1 {
		return xerrors.Errorf("empty key")
	}

	if err := st.valueHash.IsValid(nil); err != nil {
		return err
	}

	if err := st.previousBlock.IsValid(nil); err != nil {
		return err
	}

	if st.currentBlock != nil {
		if err := st.currentBlock.IsValid(nil); err != nil {
			return err
		}
	}

	for i := range st.operations {
		if err := st.operations[i].IsValid(nil); err != nil {
			return err
		}
	}

	return nil
}

func (st StateV0) Hint() hint.Hint {
	return StateV0Hint
}

func (st StateV0) Hash() valuehash.Hash {
	return st.h
}

func (st *StateV0) SetHash(h valuehash.Hash) error {
	if err := h.IsValid(nil); err != nil {
		return err
	}

	st.h = h

	return nil
}

func (st StateV0) GenerateHash() valuehash.Hash {
	be := make([][]byte, len(st.operations))
	be = append(
		be,
		[]byte(st.key),
		st.previousBlock.Bytes(),
		st.valueHash.Bytes(),
	)

	for _, oi := range st.operations {
		be = append(be, oi.Bytes())
	}

	e := util.ConcatSlice(be)

	return valuehash.NewSHA256(e)
}

func (st StateV0) Key() string {
	return st.key
}

func (st StateV0) Value() interface{} {
	return st.value
}

func (st StateV0) ValueHash() valuehash.Hash {
	return st.valueHash
}

func (st *StateV0) SetValue(value interface{}, valueHash valuehash.Hash) error {
	st.value = value
	st.valueHash = valueHash

	return nil
}

func (st StateV0) PreviousBlock() valuehash.Hash {
	return st.previousBlock
}

func (st *StateV0) SetPreviousBlock(h valuehash.Hash) error {
	if err := h.IsValid(nil); err != nil {
		return err
	}

	st.previousBlock = h

	return nil
}

func (st StateV0) CurrentBlock() valuehash.Hash {
	return st.currentBlock
}

func (st *StateV0) SetCurrentBlock(h valuehash.Hash) error {
	if err := h.IsValid(nil); err != nil {
		return err
	}

	st.currentBlock = h

	return nil
}

func (st StateV0) Operations() []OperationInfo {
	return st.operations
}

func (st *StateV0) AddOperationInfo(opi OperationInfo) error {
	if err := opi.IsValid(nil); err != nil {
		return err
	}

	st.operations = append(st.operations, opi)

	return nil
}

type OperationInfoV0 struct {
	oh valuehash.Hash
	sh valuehash.Hash
	op operation.Operation
}

func NewOperationInfoV0(op operation.Operation, sh valuehash.Hash) OperationInfoV0 {
	return OperationInfoV0{
		oh: op.Hash(),
		sh: sh,
		op: op,
	}
}

func (oi OperationInfoV0) Hint() hint.Hint {
	return OperationInfoV0Hint
}

func (oi OperationInfoV0) IsValid([]byte) error {
	if err := oi.oh.IsValid(nil); err != nil {
		return err
	}

	if err := oi.sh.IsValid(nil); err != nil {
		return err
	}

	return nil
}

func (oi OperationInfoV0) Operation() valuehash.Hash {
	return oi.oh
}

func (oi OperationInfoV0) RawOperation() operation.Operation {
	return oi.op
}

func (oi OperationInfoV0) Seal() valuehash.Hash {
	return oi.sh
}

func (oi OperationInfoV0) Bytes() []byte {
	return util.ConcatSlice([][]byte{
		oi.oh.Bytes(),
		oi.sh.Bytes(),
	})
}
