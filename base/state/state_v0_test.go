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
	st, err := NewStateV0(
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
	opi := NewOperationInfoV0(op, valuehash.RandomSHA256())

	t.NoError(st.AddOperationInfo(opi))

	t.Equal(1, len(st.Operations()))
	t.True(st.Operations()[0].Operation().Equal(op.Hash()))
	t.True(st.Operations()[0].Seal().Equal(opi.Seal()))

	t.NoError(st.AddOperationInfo(opi))
	t.Equal(1, len(st.Operations()))

	t.Equal(1, len(st.Operations()))
	t.True(st.Operations()[0].Operation().Equal(op.Hash()))
	t.True(st.Operations()[0].Seal().Equal(opi.Seal()))

	t.Equal(1, len(st.opcache))

	// NOTE SetCurrentBlock() will initialize opcache
	t.NoError(st.SetCurrentBlock(base.Height(3), valuehash.RandomSHA256()))
	t.Empty(st.opcache)
}

func TestStateV0(t *testing.T) {
	suite.Run(t, new(testStateV0))
}
