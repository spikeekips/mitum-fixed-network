package state

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
)

type testStateV0 struct {
	suite.Suite
}

func (t *testStateV0) TestDuplicatedOperation() {
	value, _ := NewBytesValue(util.UUID().Bytes())
	st, err := NewStateV0(
		util.UUID().String(),
		value,
		base.NilHeight,
	)
	t.NoError(err)
	t.Implements((*State)(nil), st)

	stu := NewStateUpdater(st)

	op, err := operation.NewKVOperation(
		key.MustNewBTCPrivatekey(),
		util.UUID().Bytes(),
		stu.Key(),
		value.Interface().([]byte),
		nil,
	)

	t.NoError(stu.AddOperation(op.Hash()))

	t.Equal(1, len(stu.Operations()))
	t.True(stu.Operations()[0].Equal(op.Hash()))

	t.NoError(stu.AddOperation(op.Hash()))
	t.Equal(1, len(stu.Operations()))

	t.Equal(1, len(stu.Operations()))
	t.True(stu.Operations()[0].Equal(op.Hash()))

	t.Equal(1, len(stu.opcache))

	// NOTE SetCurrentBlock() will initialize opcache
	stu.SetHeight(base.Height(3))
	t.Empty(stu.opcache)
}

func (t *testStateV0) TestMerge() {
	k := util.UUID().String()

	v0, _ := NewBytesValue(util.UUID().Bytes())
	s0, err := NewStateV0(k, v0, base.NilHeight)
	t.NoError(err)

	v1, _ := NewBytesValue(util.UUID().Bytes())
	s1, err := NewStateV0(k, v1, base.NilHeight)
	t.NoError(err)

	ns, err := s0.Merge(s1)
	t.NoError(err)

	t.True(ns.Value().Equal(s1.Value()))
}

func (t *testStateV0) TestMergeNil() {
	k := util.UUID().String()

	{ // not nil -> nil
		s0, err := NewStateV0(k, nil, base.NilHeight)
		t.NoError(err)

		v1, _ := NewBytesValue(util.UUID().Bytes())
		s1, err := NewStateV0(k, v1, base.NilHeight)
		t.NoError(err)

		ns, err := s0.Merge(s1)
		t.NoError(err)

		t.True(ns.Value().Equal(s1.Value()))
	}

	{ // nil -> not nil
		v0, _ := NewBytesValue(util.UUID().Bytes())
		s0, err := NewStateV0(k, v0, base.NilHeight)
		t.NoError(err)

		s1, err := NewStateV0(k, nil, base.NilHeight)
		t.NoError(err)

		ns, err := s0.Merge(s1)
		t.NoError(err)

		t.True(ns.Value().Equal(s0.Value()))
	}

	{ // nil -> nil
		s0, err := NewStateV0(k, nil, base.NilHeight)
		t.NoError(err)

		s1, err := NewStateV0(k, nil, base.NilHeight)
		t.NoError(err)

		ns, err := s0.Merge(s1)
		t.NoError(err)

		t.Nil(ns.Value())
	}
}

func (t *testStateV0) TestMergeDifferentKey() {
	v0, _ := NewBytesValue(util.UUID().Bytes())
	s0, err := NewStateV0(util.UUID().String(), v0, base.NilHeight)
	t.NoError(err)

	v1, _ := NewBytesValue(util.UUID().Bytes())
	s1, err := NewStateV0(util.UUID().String(), v1, base.NilHeight)
	t.NoError(err)

	_, err = s0.Merge(s1)
	t.Contains(err.Error(), "different key found during merging")
}

func (t *testStateV0) TestMergeUpdater() {
	k := util.UUID().String()

	v0, _ := NewBytesValue(util.UUID().Bytes())
	st0, err := NewStateV0(k, v0, base.NilHeight)
	t.NoError(err)
	s0 := NewStateUpdater(st0)

	v1, _ := NewBytesValue(util.UUID().Bytes())
	s1, err := NewStateV0(k, v1, base.NilHeight)
	t.NoError(err)

	t.NoError(s0.Merge(s1))

	t.True(v0.Equal(s0.Value()))
}

func TestStateV0(t *testing.T) {
	suite.Run(t, new(testStateV0))
}
