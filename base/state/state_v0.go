package state

import (
	"bytes"
	"sort"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	StateV0Type = hint.Type("stete")
	StateV0Hint = hint.NewHint(StateV0Type, "v0.0.1")
)

type StateV0 struct {
	h              valuehash.Hash
	key            string
	value          Value
	height         base.Height
	previousHeight base.Height
	operations     []valuehash.Hash
}

func NewStateV0(key string, value Value, height base.Height) (StateV0, error) {
	if err := IsValidKey(key); err != nil {
		return StateV0{}, err
	}

	return StateV0{
		key:            key,
		value:          value,
		previousHeight: base.NilHeight,
		height:         height,
	}, nil
}

func (st StateV0) IsValid([]byte) error {
	if err := IsValidKey(st.key); err != nil {
		return err
	}

	if st.h != nil && st.h.IsEmpty() {
		return isvalid.InvalidError.Errorf("empty hash found")
	}

	if err := st.value.IsValid(nil); err != nil {
		return err
	}

	if !st.previousHeight.IsEmpty() {
		if err := st.previousHeight.IsValid(nil); err != nil {
			return err
		}
	}

	if !st.height.IsEmpty() {
		if err := st.height.IsValid(nil); err != nil {
			return err
		}
	}

	vs := make([]isvalid.IsValider, len(st.operations))
	for i := range st.operations {
		vs[i] = st.operations[i]
	}

	return isvalid.Check(nil, false, vs...)
}

func (StateV0) Hint() hint.Hint {
	return StateV0Hint
}

func (st StateV0) Hash() valuehash.Hash {
	return st.h
}

func (st StateV0) SetHash(h valuehash.Hash) (State, error) {
	if err := h.IsValid(nil); err != nil {
		return nil, err
	}

	st.h = h

	return st, nil
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
	if st.previousHeight != base.NilHeight {
		pbb = st.previousHeight.Bytes()
	}

	return valuehash.NewSHA256(util.ConcatBytesSlice(
		[]byte(st.key),
		st.value.Hash().Bytes(),
		pbb,
		util.ConcatBytesSlice(opb...),
	))
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

func (st StateV0) PreviousHeight() base.Height {
	return st.previousHeight
}

func (st StateV0) SetPreviousHeight(h base.Height) (State, error) {
	st.previousHeight = h

	return st, nil
}

func (st StateV0) Height() base.Height {
	return st.height
}

func (st StateV0) SetHeight(h base.Height) State {
	if st.height != h {
		st.previousHeight = st.height
		st.height = h
	}

	return st
}

func (st StateV0) Operations() []valuehash.Hash {
	return st.operations
}

func (st StateV0) SetOperation(ops []valuehash.Hash) State {
	st.operations = ops

	return st
}

func (st StateV0) Merge(source State) (State, error) {
	if st.Key() != source.Key() {
		return nil, errors.Errorf("different key found during merging")
	}

	if source.Value() == nil {
		return st, nil
	} else if st.Value() != nil && st.Value().Equal(source.Value()) {
		return st, nil
	}

	st.value = source.Value()

	return st, nil
}

func (st StateV0) Clear() State {
	st.operations = nil

	return st
}
