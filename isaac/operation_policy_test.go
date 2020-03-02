package isaac

import (
	"testing"

	"github.com/spikeekips/mitum/key"
	"github.com/stretchr/testify/suite"
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
	spo, err := NewSetPolicyOperationV0(t.pk, token, nil)
	t.NoError(err)

	{
		err = spo.IsValid(nil)
		t.Contains(err.Error(), "NumberOfActingSuffrageNodes")

		spo.NumberOfActingSuffrageNodes = 1
	}

	{
		err = spo.IsValid(nil)
		t.Contains(err.Error(), "zero total found")

		threshold, err := NewThreshold(3, 100)
		t.NoError(err)
		spo.Threshold = threshold
	}

	t.NoError(spo.IsValid(nil))
	t.NoError(IsValidOperation(spo, nil))

	t.Implements((*Operation)(nil), spo)
	t.NotNil(spo.Hash())
}

func (t *testSetPolicyOperation) TestNilSigner() {
	_, err := NewSetPolicyOperationV0(nil, []byte("a"), nil)
	t.Contains(err.Error(), "empty privatekey")
}

func (t *testSetPolicyOperation) TestBadToken() {
	{ // nil
		spo, err := NewSetPolicyOperationV0(t.pk, nil, nil)
		t.NoError(err)
		err = spo.IsValid(nil)
		t.Contains(err.Error(), "empty token")
	}

	{ // zero
		spo, err := NewSetPolicyOperationV0(t.pk, []byte{}, nil)
		t.NoError(err)
		err = spo.IsValid(nil)
		t.Contains(err.Error(), "empty token")
	}

	{ // over MaxTokenSize
		token := [MaxTokenSize + 1]byte{}
		spo, err := NewSetPolicyOperationV0(t.pk, token[:], nil)
		t.NoError(err)
		err = spo.IsValid(nil)
		t.Contains(err.Error(), "token size too large")
	}
}

func TestSetPolicyOperation(t *testing.T) {
	suite.Run(t, new(testSetPolicyOperation))
}
