package operation

import (
	"fmt"
	"testing"

	"github.com/spikeekips/mitum/base/key"
	"github.com/stretchr/testify/suite"
)

type testSeal struct {
	suite.Suite

	pk key.Privatekey
}

func (t *testSeal) SetupSuite() {
	t.pk = key.NewBasePrivatekey()
}

func (t *testSeal) TestSign() {
	token := []byte("this-is-token")

	var ops []Operation
	for i := 0; i < 3; i++ {
		op, err := NewKVOperation(t.pk, token, fmt.Sprintf("d-%d", i), []byte(fmt.Sprintf("v-%d", i)), nil)
		t.NoError(err)

		ops = append(ops, op)
	}
	sl, err := NewBaseSeal(t.pk, ops, nil)
	t.NoError(err)

	t.Implements((*Seal)(nil), sl)
	t.NoError(sl.IsValid(nil))

	t.Equal(3, len(sl.Operations()))
}

func TestSeal(t *testing.T) {
	suite.Run(t, new(testSeal))
}
