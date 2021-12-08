package base

import (
	"errors"
	"testing"
	"time"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/stretchr/testify/suite"
)

type testBaseBallotFactSign struct {
	suite.Suite
}

func (t *testBaseBallotFactSign) TestNew() {
	priv := key.NewBasePrivatekey()
	pub := priv.Publickey()

	fs := NewBaseBallotFactSign(RandomStringAddress(), pub, time.Now(), key.NewSignatureFromString("findme"))
	t.NoError(fs.IsValid(nil))
}

func (t *testBaseBallotFactSign) TestEmptyNode() {
	priv := key.NewBasePrivatekey()
	pub := priv.Publickey()

	fs := NewBaseBallotFactSign(nil, pub, time.Now(), key.NewSignatureFromString("findme"))
	err := fs.IsValid(nil)
	t.True(errors.Is(err, isvalid.InvalidError))
}

func TestBaseBallotFactSign(t *testing.T) {
	suite.Run(t, new(testBaseBallotFactSign))
}
