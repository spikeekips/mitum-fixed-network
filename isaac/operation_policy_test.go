package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
)

type testSetPolicyOperation struct {
	suite.Suite

	pk key.BTCPrivatekey
}

func (t *testSetPolicyOperation) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testSetPolicyOperation) TestNew() {
	token := []byte("findme")

	{
		policies := DefaultPolicy()
		policies.NumberOfActingSuffrageNodes = 0

		spo, err := NewSetPolicyOperationV0(t.pk, token, policies, nil)
		t.NoError(err)

		err = spo.IsValid(nil)
		t.Contains(err.Error(), "NumberOfActingSuffrageNodes")
	}

	{
		policies := DefaultPolicy()

		policies.Threshold.Total = 0

		spo, err := NewSetPolicyOperationV0(t.pk, token, policies, nil)
		t.NoError(err)

		err = spo.IsValid(nil)
		t.Contains(err.Error(), "zero total found")
	}

	{
		spo, err := NewSetPolicyOperationV0(t.pk, token, DefaultPolicy(), nil)
		t.NoError(err)

		t.NoError(spo.IsValid(nil))
		t.NoError(operation.IsValidOperation(spo, nil))

		t.Implements((*operation.Operation)(nil), spo)
		t.NotNil(spo.Hash())
	}
}

func (t *testSetPolicyOperation) TestNilSigner() {
	_, err := NewSetPolicyOperationV0(nil, []byte("a"), DefaultPolicy(), nil)
	t.Contains(err.Error(), "empty privatekey")
}

func (t *testSetPolicyOperation) TestBadToken() {
	{ // nil
		spo, err := NewSetPolicyOperationV0(t.pk, nil, DefaultPolicy(), nil)
		t.NoError(err)
		err = spo.IsValid(nil)
		t.Contains(err.Error(), "empty token")
	}

	{ // zero
		spo, err := NewSetPolicyOperationV0(t.pk, []byte{}, DefaultPolicy(), nil)
		t.NoError(err)
		err = spo.IsValid(nil)
		t.Contains(err.Error(), "empty token")
	}

	{ // over MaxTokenSize
		token := [operation.MaxTokenSize + 1]byte{}
		spo, err := NewSetPolicyOperationV0(t.pk, token[:], DefaultPolicy(), nil)
		t.NoError(err)
		err = spo.IsValid(nil)
		t.Contains(err.Error(), "token size too large")
	}
}

func TestSetPolicyOperation(t *testing.T) {
	suite.Run(t, new(testSetPolicyOperation))
}
