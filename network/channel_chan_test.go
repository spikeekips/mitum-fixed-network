package network

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/valuehash"
)

type dummySeal struct {
	pk       key.Privatekey
	h        valuehash.Hash
	bodyHash valuehash.Hash
}

func newDummySeal(pk key.Privatekey) dummySeal {
	return dummySeal{
		pk:       pk,
		h:        valuehash.RandomSHA256(),
		bodyHash: valuehash.RandomSHA256(),
	}
}

func (ds dummySeal) IsValid([]byte) error {
	return nil
}

func (ds dummySeal) Hint() hint.Hint {
	return hint.MustHint(hint.Type([2]byte{0xff, 0x30}), "0.1")
}

func (ds dummySeal) Hash() valuehash.Hash {
	return ds.h
}

func (ds dummySeal) GenerateHash([]byte) (valuehash.Hash, error) {
	return ds.h, nil
}

func (ds dummySeal) BodyHash() valuehash.Hash {
	return ds.bodyHash
}

func (ds dummySeal) GenerateBodyHash([]byte) (valuehash.Hash, error) {
	return ds.bodyHash, nil
}

func (ds dummySeal) Signer() key.Publickey {
	return ds.pk.Publickey()
}

func (ds dummySeal) Signature() key.Signature {
	return key.Signature([]byte("showme"))
}

func (ds dummySeal) SignedAt() time.Time {
	return localtime.Now()
}

type testChanChannel struct {
	suite.Suite

	pk key.BTCPrivatekey
}

func (t *testChanChannel) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testChanChannel) TestSendReceive() {
	gs := NewChanChannel(0, nil)

	sl := newDummySeal(t.pk)
	t.NoError(gs.SendSeal(sl))

	rsl := <-gs.ReceiveSeal()

	t.True(sl.Hash().Equal(rsl.Hash()))
}

func (t *testChanChannel) TestSealHandler() {
	gs := NewChanChannel(0, func(sl seal.Seal) (seal.Seal, error) {
		return nil, xerrors.Errorf("invalid seal found")
	})

	sl := newDummySeal(t.pk)
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
