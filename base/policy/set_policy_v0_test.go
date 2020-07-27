package policy

import (
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testSetPolicyV0 struct {
	suite.Suite
	priv key.Privatekey
}

func (t *testSetPolicyV0) SetupSuite() {
	t.priv = key.MustNewBTCPrivatekey()
}

func (t *testSetPolicyV0) TestNewFact() {
	po := NewPolicyV0(base.ThresholdRatio(33), 3, 6, 9)
	t.NoError(po.IsValid(nil))

	fact := NewSetPolicyFactV0(po, util.UUID().Bytes())
	t.NoError(fact.IsValid(nil))

	t.Implements((*base.Fact)(nil), fact)
}

func (t *testSetPolicyV0) TestNew() {
	po := NewPolicyV0(base.ThresholdRatio(33), 3, 6, 9)
	t.NoError(po.IsValid(nil))

	networkID := util.UUID().Bytes()

	spo, err := NewSetPolicyV0(po, util.UUID().Bytes(), t.priv, networkID)
	t.NoError(err)

	t.Implements((*operation.Operation)(nil), spo)

	err = spo.IsValid(nil)
	t.True(xerrors.Is(err, key.SignatureVerificationFailedError))

	t.NoError(spo.IsValid(networkID))
}

func TestSetPolicyV0(t *testing.T) {
	s := new(testSetPolicyV0)

	suite.Run(t, s)
}
