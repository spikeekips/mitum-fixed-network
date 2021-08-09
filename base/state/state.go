package state

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

type State interface {
	isvalid.IsValider
	hint.Hinter
	valuehash.Hasher
	util.Byter
	Key() string
	Value() Value
	SetValue(Value) (State, error)
	SetHash(valuehash.Hash) (State, error)
	Height() base.Height
	SetHeight(base.Height) State
	PreviousHeight() base.Height
	SetPreviousHeight(base.Height) (State, error)
	Operations() []valuehash.Hash
	SetOperation([]valuehash.Hash) State
	GenerateHash() valuehash.Hash
	Merge(State) (State, error)
	Clear() State
}

type StateUpdater struct {
	sync.RWMutex
	State
	opcache    map[string]struct{}
	orig       State
	height     base.Height
	operations []valuehash.Hash
}

func NewStateUpdater(st State) *StateUpdater {
	stu := StateUpdaterPoolGet()
	stu.RWMutex = sync.RWMutex{}
	stu.State = st
	stu.opcache = map[string]struct{}{}
	stu.orig = st
	stu.height = base.NilHeight
	stu.operations = nil

	return stu
}

func (stu *StateUpdater) SetHash(h valuehash.Hash) error {
	stu.Lock()
	defer stu.Unlock()

	if err := h.IsValid(nil); err != nil {
		return err
	}

	st, err := stu.State.SetHash(h)
	if err != nil {
		return err
	}

	stu.State = st

	return nil
}

func (stu *StateUpdater) SetValue(value Value) error {
	stu.Lock()
	defer stu.Unlock()

	st, err := stu.State.SetValue(value)
	if err != nil {
		return err
	}

	stu.State = st

	return nil
}

func (stu *StateUpdater) SetHeight(h base.Height) *StateUpdater {
	stu.Lock()
	defer stu.Unlock()

	stu.height = h
	stu.opcache = map[string]struct{}{}

	return stu
}

func (stu *StateUpdater) Operations() []valuehash.Hash {
	return stu.operations
}

func (stu *StateUpdater) AddOperation(op valuehash.Hash) error {
	stu.Lock()
	defer stu.Unlock()

	oh := op.String()
	if _, found := stu.opcache[oh]; found {
		return nil
	}

	stu.opcache[oh] = struct{}{}

	if err := op.IsValid(nil); err != nil {
		return err
	}

	stu.operations = append(stu.operations, op)

	return nil
}

func (stu *StateUpdater) Merge(source State) error {
	stu.Lock()
	defer stu.Unlock()

	if stu.Key() != source.Key() {
		return errors.Errorf("different key found during merging")
	} else if ns, err := source.Merge(stu.State); err != nil {
		return err
	} else {
		stu.State = ns

		return nil
	}
}

func (stu *StateUpdater) Reset() error {
	stu.Lock()
	defer stu.Unlock()

	stu.State = stu.orig

	return nil
}

func (stu *StateUpdater) GetState() State {
	return stu.State.SetHeight(stu.height).SetOperation(stu.operations)
}
