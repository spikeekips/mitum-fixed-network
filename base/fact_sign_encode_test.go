package base

import (
	"testing"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/stretchr/testify/suite"
)

type testFactSignEncoding struct {
	suite.Suite

	pk   key.Privatekey
	encs *encoder.Encoders
	enc  encoder.Encoder
}

func (t *testFactSignEncoding) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()

	t.encs = encoder.NewEncoders()
	_ = t.encs.AddEncoder(t.enc)

	_ = t.encs.TestAddHinter(key.BTCPublickeyHinter)
	_ = t.encs.TestAddHinter(BaseFactSignHinter)
}

func (t *testFactSignEncoding) TestMarshal() {
	fs := NewBaseFactSign(t.pk.Publickey(), util.UUID().Bytes())
	t.NoError(fs.IsValid(nil))

	b, err := t.enc.Marshal(fs)
	t.NoError(err)

	hinter, err := t.enc.Decode(b)
	t.NoError(err)
	t.IsType(hinter, BaseFactSign{})

	ufs := hinter.(FactSign)

	t.True(fs.Signer().Equal(ufs.Signer()))
	t.Equal(fs.Signature(), ufs.Signature())
	t.True(localtime.Equal(fs.SignedAt(), ufs.SignedAt()))
}

func TestFactSignEncodingJSON(t *testing.T) {
	s := new(testFactSignEncoding)
	s.enc = jsonenc.NewEncoder()

	suite.Run(t, s)
}

func TestFactSignEncodingBSON(t *testing.T) {
	s := new(testFactSignEncoding)
	s.enc = bsonenc.NewEncoder()

	suite.Run(t, s)
}
