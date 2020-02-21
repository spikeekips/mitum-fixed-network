package network

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/seal"
)

type testChanChannel struct {
	suite.Suite

	pk key.BTCPrivatekey
}

func (t *testChanChannel) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testChanChannel) TestSendReceive() {
	gs := NewChanChannel(0, nil)

	sl := seal.NewDummySeal(t.pk)
	t.NoError(gs.SendSeal(sl))

	rsl := <-gs.ReceiveSeal()

	t.True(sl.Hash().Equal(rsl.Hash()))
}

func (t *testChanChannel) TestSealHandler() {
	gs := NewChanChannel(0, func(sl seal.Seal) (seal.Seal, error) {
		return nil, xerrors.Errorf("invalid seal found")
	})

	sl := seal.NewDummySeal(t.pk)
	t.NoError(gs.SendSeal(sl))

	select {
	case <-time.After(time.Millisecond * 10):
		break
	case <-gs.ReceiveSeal():
		t.Error(xerrors.Errorf("seal should be ignored"))
	}
}

func TestChanChannel(t *testing.T) {
	suite.Run(t, new(testChanChannel))
}
