package network

import (
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/stretchr/testify/suite"
)

type testHandoverSealV0 struct {
	suite.Suite
	pk   key.Privatekey
	encs *encoder.Encoders
	enc  encoder.Encoder
}

func (t *testHandoverSealV0) SetupSuite() {
	t.pk = key.NewBasePrivatekey()

	t.encs = encoder.NewEncoders()
	_ = t.encs.AddEncoder(t.enc)

	_ = t.encs.TestAddHinter(key.BasePublickey{})
	_ = t.encs.TestAddHinter(NilConnInfoHinter)
	_ = t.encs.TestAddHinter(base.StringAddressHinter)
	_ = t.encs.TestAddHinter(StartHandoverSealV0Hinter)
	_ = t.encs.TestAddHinter(PingHandoverSealV0Hinter)
	_ = t.encs.TestAddHinter(EndHandoverSealV0Hinter)
}

func (t *testHandoverSealV0) testSign(ht hint.Hint) {
	sl, err := NewHandoverSealV0(ht, t.pk, base.RandomStringAddress(), NewNilConnInfo("showme"), nil)
	t.NoError(err)

	var raw []byte
	raw, err = t.enc.Marshal(sl)
	t.NoError(err)

	hinter, err := t.enc.Decode(raw)
	t.NoError(err)

	usl, ok := hinter.(HandoverSealV0)
	t.True(ok)

	t.NoError(usl.IsValid(nil))

	t.True(sl.Hint().Equal(usl.Hint()))
	t.True(sl.Hash().Equal(usl.Hash()))
	t.True(sl.ci.Equal(usl.ci))
}

func (t *testHandoverSealV0) TestSeals() {
	t.Run("start-handover-seal", func() {
		t.testSign(StartHandoverSealV0Hint)
	})
	t.Run("ping-handover-seal", func() {
		t.testSign(PingHandoverSealV0Hint)
	})
	t.Run("end-handover-seal", func() {
		t.testSign(EndHandoverSealV0Hint)
	})
}

func TestHandoverSealV0JSON(t *testing.T) {
	b := new(testHandoverSealV0)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestHandoverSealV0BSON(t *testing.T) {
	b := new(testHandoverSealV0)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
