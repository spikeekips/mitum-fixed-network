package state

import (
	"sync"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	StateV0Type = hint.MustNewType(0x01, 0x60, "stete-v0")
	StateV0Hint = hint.MustHint(StateV0Type, "0.0.1")
)

type StateV0 struct {
	h             valuehash.Hash
	key           string
	value         Value
	previousBlock valuehash.Hash
	operations    []valuehash.Hash
	currentHeight base.Height
	currentBlock  valuehash.Hash
}

func (st StateV0) IsValid([]byte) error {
	if err := IsValidKey(st.key); err != nil {
		return err
	}

	if st.h != nil && st.h.Empty() {
		return xerrors.Errorf("empty hash found")
	}

	if st.currentBlock != nil && st.currentBlock.Empty() {
		return xerrors.Errorf("empty current block hash found")
	}

	if err := st.value.IsValid(nil); err != nil {
		return err
	}

	if st.previousBlock != nil {
		if err := st.previousBlock.IsValid(nil); err != nil {
			return err
		}
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

func (st StateV0) GenerateHash() valuehash.Hash {
	be := make([][]byte, len(st.operations))
	be = append(
		be,
		[]byte(st.key),
		st.value.Hash().Bytes(),
	)

	if st.previousBlock != nil {
		be = append(be, st.previousBlock.Bytes())
	}

	for _, oi := range st.operations {
		be = append(be, oi.Bytes())
	}

	return valuehash.NewSHA256(util.ConcatBytesSlice(be...))
}

func (st StateV0) Key() string {
	return st.key
}

func (st StateV0) Value() Value {
	return st.value
}

func (st StateV0) PreviousBlock() valuehash.Hash {
	return st.previousBlock
}

func (st StateV0) Height() base.Height {
	return st.currentHeight
}

func (st StateV0) CurrentBlock() valuehash.Hash {
	return st.currentBlock
}

func (st StateV0) Operations() []valuehash.Hash {
	return st.operations
}

type StateV0Updater struct {
	StateV0
	sync.RWMutex
	opcache   map[string]struct{}
	origValue Value
}

func NewStateV0Updater(key string, value Value, previousBlock valuehash.Hash) (*StateV0Updater, error) {
	if err := IsValidKey(key); err != nil {
		return nil, err
	}

	return &StateV0Updater{
		StateV0: StateV0{
			key:           key,
			value:         value,
			previousBlock: previousBlock,
		},
		opcache:   map[string]struct{}{},
		origValue: value,
	}, nil
}

func (stu *StateV0Updater) State() StateV0 {
	stu.RLock()
	defer stu.RUnlock()

	return stu.StateV0
}

func (stu *StateV0Updater) OriginalValue() Value {
	return stu.origValue
}

func (stu *StateV0Updater) Key() string {
	stu.RLock()
	defer stu.RUnlock()

	return stu.StateV0.key
}

func (stu *StateV0Updater) Hash() valuehash.Hash {
	stu.RLock()
	defer stu.RUnlock()

	return stu.h
}

func (stu *StateV0Updater) SetHash(h valuehash.Hash) error {
	stu.Lock()
	defer stu.Unlock()

	if err := h.IsValid(nil); err != nil {
		return err
	}

	stu.h = h

	return nil
}

func (stu *StateV0Updater) Value() Value {
	stu.RLock()
	defer stu.RUnlock()

	return stu.StateV0.value
}

func (stu *StateV0Updater) SetValue(value Value) error {
	stu.Lock()
	defer stu.Unlock()

	stu.StateV0.value = value

	return nil
}

func (stu *StateV0Updater) PreviousBlock() valuehash.Hash {
	stu.RLock()
	defer stu.RUnlock()

	return stu.previousBlock
}

func (stu *StateV0Updater) SetPreviousBlock(h valuehash.Hash) error {
	stu.Lock()
	defer stu.Unlock()

	if err := h.IsValid(nil); err != nil {
		return err
	}

	stu.previousBlock = h

	return nil
}

func (stu *StateV0Updater) CurrentBlock() valuehash.Hash {
	stu.RLock()
	defer stu.RUnlock()

	return stu.currentBlock
}

func (stu *StateV0Updater) SetCurrentBlock(height base.Height, h valuehash.Hash) error {
	stu.Lock()
	defer stu.Unlock()

	if err := h.IsValid(nil); err != nil {
		return err
	}

	stu.currentHeight = height
	stu.currentBlock = h

	stu.opcache = map[string]struct{}{}

	return nil
}

func (stu *StateV0Updater) Operations() []valuehash.Hash {
	stu.RLock()
	defer stu.RUnlock()

	return stu.operations
}

func (stu *StateV0Updater) AddOperation(op valuehash.Hash) error {
	stu.Lock()
	defer stu.Unlock()

	if err := op.IsValid(nil); err != nil {
		return err
	}

	oh := op.String()
	if _, found := stu.opcache[oh]; found {
		return nil
	} else {
		stu.opcache[oh] = struct{}{}
	}

	stu.operations = append(stu.operations, op)

	return nil
}
