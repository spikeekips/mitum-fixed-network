package channetwork

import (
	"context"
	"testing"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testNetworkChanChannel struct {
	suite.Suite

	pk key.Privatekey
}

func (t *testNetworkChanChannel) SetupSuite() {
	t.pk = key.NewBasePrivatekey()
}

func (t *testNetworkChanChannel) TestSendReceive() {
	gs := NewChannel(0, network.NewNilConnInfo("showme"))
	t.Implements((*network.Channel)(nil), gs)

	sl := seal.NewDummySeal(t.pk.Publickey())
	go func() {
		_ = gs.SendSeal(context.TODO(), nil, sl)
	}()

	rsl := <-gs.ReceiveSeal()

	t.True(sl.Hash().Equal(rsl.Hash()))
}

func (t *testNetworkChanChannel) TestGetStagedOperation() {
	gs := NewChannel(0, network.NewNilConnInfo("showme"))

	op, err := operation.NewKVOperation(t.pk, util.UUID().Bytes(), util.UUID().String(), util.UUID().Bytes(), nil)
	t.NoError(err)

	gs.SetGetStagedOperationsHandler(func(hs []valuehash.Hash) ([]operation.Operation, error) {
		return []operation.Operation{op}, nil
	})

	ops, err := gs.StagedOperations(context.TODO(), []valuehash.Hash{op.Fact().Hash()})
	t.NoError(err)
	t.Equal(1, len(ops))

	t.True(op.Hash().Equal(ops[0].Hash()))
}

func TestNetworkChanChannel(t *testing.T) {
	suite.Run(t, new(testNetworkChanChannel))
}
