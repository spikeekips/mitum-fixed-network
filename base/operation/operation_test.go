package operation

import (
	"testing"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/stretchr/testify/suite"
)

type testOperation struct {
	suite.Suite

	pk key.Privatekey
}

func (t *testOperation) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testOperation) TestLastSignedAt() {
	token := []byte("this-is-token")

	networkID := util.UUID().Bytes()
	op, err := NewKVOperation(t.pk, token, util.UUID().String(), util.UUID().Bytes(), networkID)
	t.NoError(err)

	var lastfs FactSign
	var bo BaseOperation
	for i := 0; i < 3; i++ {
		pk := key.MustNewBTCPrivatekey()

		sig, err := pk.Sign(util.ConcatBytesSlice(op.Fact().Hash().Bytes(), networkID))
		t.NoError(err)

		fs := NewBaseFactSign(pk.Publickey(), sig)
		lastfs = fs

		nop, err := op.AddFactSigns(fs)
		t.NoError(err)
		bo = nop.(BaseOperation)
	}

	t.Equal(lastfs.SignedAt(), bo.LastSignedAt())
}

func TestOperation(t *testing.T) {
	suite.Run(t, new(testOperation))
}
