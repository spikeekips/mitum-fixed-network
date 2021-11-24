package operation

import (
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/stretchr/testify/suite"
)

type testOperation struct {
	suite.Suite

	pk key.Privatekey
}

func (t *testOperation) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testOperation) TestAddFactSign() {
	token := []byte("this-is-token")

	networkID := util.UUID().Bytes()
	op, err := NewKVOperation(t.pk, token, util.UUID().String(), util.UUID().Bytes(), networkID)
	t.NoError(err)

	bo := op.BaseOperation
	var privs []key.Privatekey
	for i := 0; i < 3; i++ {
		priv := key.MustNewBTCPrivatekey()
		privs = append(privs, priv)

		sig, err := priv.Sign(util.ConcatBytesSlice(op.Fact().Hash().Bytes(), networkID))
		t.NoError(err)

		fs := base.NewBaseFactSign(priv.Publickey(), sig)

		nop, err := bo.AddFactSigns(fs)
		t.NoError(err)
		bo = nop.(BaseOperation)
	}
	t.Equal(4, len(bo.Signs()))

	oldhash := bo.Hash()
	first := bo.Signs()[1]

	// Add already added, but new signed
	sig, err := privs[0].Sign(util.ConcatBytesSlice(op.Fact().Hash().Bytes(), networkID))
	t.NoError(err)

	fs := base.NewBaseFactSign(privs[0].Publickey(), sig)

	nop, err := bo.AddFactSigns(fs)
	t.NoError(err)
	bo = nop.(BaseOperation)
	t.Equal(4, len(bo.Signs()))
	t.False(bo.Hash().Equal(oldhash))

	nfirst := bo.Signs()[1]
	t.True(first.Signer().Equal(nfirst.Signer()))
	t.True(first.Signature().Equal(nfirst.Signature()))
	t.False(localtime.Equal(first.SignedAt(), nfirst.SignedAt()))
}

func (t *testOperation) TestLastSignedAt() {
	token := []byte("this-is-token")

	networkID := util.UUID().Bytes()
	op, err := NewKVOperation(t.pk, token, util.UUID().String(), util.UUID().Bytes(), networkID)
	t.NoError(err)

	var lastfs base.FactSign
	var bo BaseOperation
	for i := 0; i < 3; i++ {
		pk := key.MustNewBTCPrivatekey()

		sig, err := pk.Sign(util.ConcatBytesSlice(op.Fact().Hash().Bytes(), networkID))
		t.NoError(err)

		fs := base.NewBaseFactSign(pk.Publickey(), sig)
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
