package operation

import (
	"fmt"
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/stretchr/testify/suite"
)

type testSealEncode struct {
	suite.Suite

	pk   key.Privatekey
	encs *encoder.Encoders
	enc  encoder.Encoder
}

func (t *testSealEncode) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()

	t.encs = encoder.NewEncoders()
	_ = t.encs.AddEncoder(t.enc)

	_ = t.encs.TestAddHinter(key.BTCPublickeyHinter)
	_ = t.encs.TestAddHinter(SealHinter)
	_ = t.encs.TestAddHinter(KVOperation{})
	_ = t.encs.TestAddHinter(KVOperationFact{})
	_ = t.encs.TestAddHinter(base.BaseFactSign{})
}

func (t *testSealEncode) TestSign() {
	token := []byte("this-is-token")

	var ops []Operation
	for i := 0; i < 3; i++ {
		op, err := NewKVOperation(t.pk, token, fmt.Sprintf("d-%d", i), []byte(fmt.Sprintf("v-%d", i)), nil)
		t.NoError(err)

		ops = append(ops, op)
	}
	sl, err := NewBaseSeal(t.pk, ops, nil)
	t.NoError(err)

	var raw []byte
	raw, err = t.enc.Marshal(sl)
	t.NoError(err)

	hinter, err := t.enc.Decode(raw)
	t.NoError(err)

	usl, ok := hinter.(BaseSeal)
	t.True(ok)

	t.NoError(usl.IsValid(nil))

	t.True(sl.Hash().Equal(usl.Hash()))
	t.Equal(len(sl.Operations()), len(usl.Operations()))

	for i := range sl.Operations() {
		a := sl.Operations()[i].(KVOperation)
		b := usl.Operations()[i].(KVOperation)

		t.True(a.Hash().Equal(b.Hash()))
		t.True(a.Fact().Hash().Equal(b.Fact().Hash()))
		t.Equal(a.Key(), b.Key())
		t.Equal(a.Value(), b.Value())

		t.Equal(len(a.Signs()), len(b.Signs()))
		for j := range a.Signs() {
			sa := a.Signs()[j]
			sb := b.Signs()[j]

			t.True(sa.Signer().Equal(sb.Signer()))
			t.Equal(sa.Signature(), sb.Signature())
			t.True(localtime.Equal(sa.SignedAt(), sb.SignedAt()))
		}
	}
}

func TestSealEncodeJSON(t *testing.T) {
	b := new(testSealEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestSealEncodeBSON(t *testing.T) {
	b := new(testSealEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
