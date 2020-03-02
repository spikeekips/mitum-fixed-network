package operation

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/valuehash"
)

type testSeal struct {
	suite.Suite

	pk key.BTCPrivatekey
}

func (t *testSeal) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()

	_ = hint.RegisterType(key.BTCPublickey{}.Hint().Type(), "btc-publickey")
	_ = hint.RegisterType(valuehash.SHA256{}.Hint().Type(), "sha256")
	_ = hint.RegisterType(encoder.JSONEncoder{}.Hint().Type(), "json-encoder")
	_ = hint.RegisterType(Seal{}.Hint().Type(), "operation-seal")
	_ = hint.RegisterType(KVOperation{}.Hint().Type(), "KVOperation")
	_ = hint.RegisterType(KVOperationFact{}.Hint().Type(), "KVOperation-fact")
}

func (t *testSeal) TestSign() {
	token := []byte("this-is-token")

	var ops []Operation
	for i := 0; i < 3; i++ {
		op, err := NewKVOperation(t.pk, token, fmt.Sprintf("d-%d", i), []byte(fmt.Sprintf("v-%d", i)), nil)
		t.NoError(err)

		ops = append(ops, op)
	}
	sl, err := NewSeal(t.pk, ops, nil)
	t.NoError(err)
	t.NoError(sl.IsValid(nil))

	t.Equal(3, len(sl.Operations()))
}

func TestSeal(t *testing.T) {
	suite.Run(t, new(testSeal))
}
