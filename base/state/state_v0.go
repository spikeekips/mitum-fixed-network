package state

import (
	"bytes"
	"sort"
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

func NewStateV0(key string, value Value, previousBlock valuehash.Hash) (StateV0, error) {
	if err := IsValidKey(key); err != nil {
		return StateV0{}, err
	}

	return StateV0{
		key:           key,
		value:         value,
		previousBlock: previousBlock,
	}, nil
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
	ops := st.operations
	sort.Slice(ops, func(i, j int) bool {
		return bytes.Compare(ops[i].Bytes(), ops[j].Bytes()) < 0
	})

	opb := make([][]byte, len(ops))
	for i := range ops {
		opb[i] = ops[i].Bytes()
	}

	var pbb []byte
	if st.previousBlock != nil {
		pbb = st.previousBlock.Bytes()
	}

	be := util.ConcatBytesSlice(
		[]byte(st.key),
		st.value.Hash().Bytes(),
		pbb,
		util.ConcatBytesSlice(opb...),
	)

	return valuehash.NewSHA256(be)
}

func (st StateV0) Key() string {
	return st.key
}

func (st StateV0) Value() Value {
	return st.value
}

func (st StateV0) SetValue(value Value) (State, error) {
	st.value = value

	return st, nil
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

func (st StateV0) Merge(source State) (State, error) {
	if st.Key() != source.Key() {
		return nil, xerrors.Errorf("different key found during merging")
	}

	if source.Value() == nil {
		return st, nil
	} else if st.Value() != nil && st.Value().Equal(source.Value()) {
		return st, nil
	}

	st.value = source.Value()

	return st, nil
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

func (stu *StateV0Updater) Key() string {
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

func (stu *StateV0Updater) setValue(value Value) {
	stu.StateV0.value = value
}

func (stu *StateV0Updater) SetValue(value Value) (State, error) {
	stu.Lock()
	defer stu.Unlock()

	stu.setValue(value)

	return stu, nil
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

func (stu *StateV0Updater) Merge(source State) (State, error) {
	stu.Lock()
	defer stu.Unlock()

	if stu.Key() != source.Key() {
		return nil, xerrors.Errorf("different key found during merging")
	} else if ns, err := source.Merge(stu.StateV0); err != nil {
		return nil, err
	} else {
		stu.setValue(ns.Value())
	}

	return stu.StateV0, nil
}

func (stu *StateV0Updater) Reset() {
	stu.Lock()
	defer stu.Unlock()

	stu.setValue(stu.origValue)
}
