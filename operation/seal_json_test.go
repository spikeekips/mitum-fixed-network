package operation

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

type testSealJSON struct {
	suite.Suite

	pk   key.BTCPrivatekey
	encs *encoder.Encoders
	enc  encoder.Encoder
}

func (t *testSealJSON) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()

	t.encs = encoder.NewEncoders()
	t.enc = encoder.NewJSONEncoder()
	_ = t.encs.AddEncoder(t.enc)

	_ = t.encs.AddHinter(key.BTCPublickey{})
	_ = t.encs.AddHinter(valuehash.SHA256{})
	_ = t.encs.AddHinter(Seal{})
	_ = t.encs.AddHinter(KVOperation{})
	_ = t.encs.AddHinter(KVOperationFact{})
}

func (t *testSealJSON) TestSign() {
	token := []byte("this-is-token")

	var ops []Operation
	for i := 0; i < 3; i++ {
		op, err := NewKVOperation(t.pk, token, fmt.Sprintf("d-%d", i), []byte(fmt.Sprintf("v-%d", i)), nil)
		t.NoError(err)

		ops = append(ops, op)
	}
	sl, err := NewSeal(t.pk, ops, nil)
	t.NoError(err)

	var raw []byte
	raw, err = util.JSONMarshal(sl)
	t.NoError(err)

	hinter, err := t.enc.DecodeByHint(raw)
	t.NoError(err)

	usl, ok := hinter.(Seal)
	t.True(ok)

	t.True(sl.Hash().Equal(usl.Hash()))
	t.Equal(len(sl.Operations()), len(usl.Operations()))

	for i := range sl.Operations() {
		a := sl.Operations()[i].(KVOperation)
		b := usl.Operations()[i].(KVOperation)

		t.True(a.Hash().Equal(b.Hash()))
		t.True(a.FactHash().Equal(b.FactHash()))
		t.Equal(a.FactSignature(), b.FactSignature())
		t.Equal(a.FactSignature(), b.FactSignature())
		t.Equal(a.Key, b.Key)
		t.Equal(a.Value, b.Value)
	}
}

func TestSealJSON(t *testing.T) {
	suite.Run(t, new(testSealJSON))
}
