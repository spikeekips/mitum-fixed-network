package channetwork

import (
	"context"
	"testing"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testNetworkChanChannel struct {
	suite.Suite

	pk key.Privatekey
}

func (t *testNetworkChanChannel) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testNetworkChanChannel) TestSendReceive() {
	gs := NewChannel(0, network.NewNilConnInfo("showme"))
	t.Implements((*network.Channel)(nil), gs)

	sl := seal.NewDummySeal(t.pk)
	go func() {
		_ = gs.SendSeal(context.TODO(), sl)
	}()

	rsl := <-gs.ReceiveSeal()

	t.True(sl.Hash().Equal(rsl.Hash()))
}

func (t *testNetworkChanChannel) TestGetSeal() {
	gs := NewChannel(0, network.NewNilConnInfo("showme"))

	sl := seal.NewDummySeal(t.pk)

	gs.SetGetSealHandler(func([]valuehash.Hash) ([]seal.Seal, error) {
		return []seal.Seal{sl}, nil
	})

	gsls, err := gs.Seals(context.TODO(), []valuehash.Hash{sl.Hash()})
	t.NoError(err)
	t.Equal(1, len(gsls))

	t.True(sl.Hash().Equal(gsls[0].Hash()))
}

func TestNetworkChanChannel(t *testing.T) {
	suite.Run(t, new(testNetworkChanChannel))
}
