package network

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/valuehash"
)

type testChanChannel struct {
	suite.Suite

	pk key.BTCPrivatekey
}

func (t *testChanChannel) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testChanChannel) TestSendReceive() {
	gs := NewChanChannel(0)

	sl := seal.NewDummySeal(t.pk)
	go func() {
		t.NoError(gs.SendSeal(sl))
	}()

	rsl := <-gs.ReceiveSeal()

	t.True(sl.Hash().Equal(rsl.Hash()))
}

func (t *testChanChannel) TestGetSeal() {
	gs := NewChanChannel(0)

	sl := seal.NewDummySeal(t.pk)

	gs.SetGetSealHandler(func([]valuehash.Hash) ([]seal.Seal, error) {
		return []seal.Seal{sl}, nil
	})

	gsls, err := gs.Seals([]valuehash.Hash{sl.Hash()})
	t.NoError(err)
	t.Equal(1, len(gsls))

	t.True(sl.Hash().Equal(gsls[0].Hash()))
}

func TestChanChannel(t *testing.T) {
	suite.Run(t, new(testChanChannel))
}
