package state

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testStateV0 struct {
	suite.Suite
}

func (t *testStateV0) TestDuplicatedOperation() {
	value, _ := NewBytesValue(util.UUID().Bytes())
	st, err := NewStateV0Updater(
		util.UUID().String(),
		value,
		nil,
	)
	t.NoError(err)

	op, err := operation.NewKVOperation(
		key.MustNewBTCPrivatekey(),
		util.UUID().Bytes(),
		st.Key(),
		value.Interface().([]byte),
		nil,
	)

	t.NoError(st.AddOperation(op.Hash()))

	t.Equal(1, len(st.Operations()))
	t.True(st.Operations()[0].Equal(op.Hash()))

	t.NoError(st.AddOperation(op.Hash()))
	t.Equal(1, len(st.Operations()))

	t.Equal(1, len(st.Operations()))
	t.True(st.Operations()[0].Equal(op.Hash()))

	t.Equal(1, len(st.opcache))

	// NOTE SetCurrentBlock() will initialize opcache
	t.NoError(st.SetCurrentBlock(base.Height(3), valuehash.RandomSHA256()))
	t.Empty(st.opcache)
}

func TestStateV0(t *testing.T) {
	suite.Run(t, new(testStateV0))
}
