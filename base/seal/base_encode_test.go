package seal

import (
	"testing"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/stretchr/testify/suite"
)

type testBaseSealEncode struct {
	suite.Suite

	pk   key.Privatekey
	encs *encoder.Encoders
	enc  encoder.Encoder
	pack func(BaseSeal) ([]byte, error)

	sealType hint.Type
	sealHint hint.Hint
}

func (t *testBaseSealEncode) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()

	t.encs = encoder.NewEncoders()
	_ = t.encs.AddEncoder(t.enc)

	t.sealType = hint.Type("seal")
	t.sealHint = hint.NewHint(t.sealType, "v0.0.1")

	sl := BaseSeal{ht: t.sealHint}

	_ = t.encs.TestAddHinter(key.BTCPublickeyHinter)
	_ = t.encs.TestAddHinter(sl)
}

func (t *testBaseSealEncode) TestSign() {
	sl, err := NewBaseSeal(t.sealHint, t.pk, nil)
	t.NoError(err)

	t.NoError(sl.IsValid(nil))

	var raw []byte
	raw, err = t.pack(sl)
	t.NoError(err)

	hinter, err := t.enc.Decode(raw)
	t.NoError(err)

	usl, ok := hinter.(BaseSeal)
	t.True(ok)
	t.NoError(usl.IsValid(nil))

	t.True(sl.Hash().Equal(usl.Hash()))
	t.True(sl.Hint().Equal(usl.Hint()))
	t.True(sl.BodyHash().Equal(usl.BodyHash()))
	t.True(sl.Signer().Equal(usl.Signer()))
	t.True(sl.Signature().Equal(usl.Signature()))
	t.True(localtime.Equal(sl.SignedAt(), usl.SignedAt()))
}

func TestBaseSealEncodeJSON(t *testing.T) {
	b := new(testBaseSealEncode)
	b.enc = jsonenc.NewEncoder()
	b.pack = func(sl BaseSeal) ([]byte, error) {
		return b.enc.Marshal(sl.JSONPacker())
	}

	suite.Run(t, b)
}

func TestBaseSealEncodeBSON(t *testing.T) {
	b := new(testBaseSealEncode)
	b.enc = bsonenc.NewEncoder()
	b.pack = func(sl BaseSeal) ([]byte, error) {
		return b.enc.Marshal(sl.BSONPacker())
	}

	suite.Run(t, b)
}
