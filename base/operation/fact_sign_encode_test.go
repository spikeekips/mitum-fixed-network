package operation

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
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

	_ = t.encs.AddHinter(key.BTCPublickey{})
	_ = t.encs.AddHinter(BaseFactSign{})
}

func (t *testFactSignEncoding) TestMarshal() {
	fs := NewBaseFactSign(t.pk.Publickey(), util.UUID().Bytes())
	t.NoError(fs.IsValid(nil))

	b, err := t.enc.Marshal(fs)
	t.NoError(err)

	hinter, err := t.enc.DecodeByHint(b)
	t.NoError(err)
	t.IsType(hinter, BaseFactSign{})

	ufs := hinter.(FactSign)

	t.True(fs.Signer().Equal(ufs.Signer()))
	t.Equal(fs.Signature(), ufs.Signature())
	t.Equal(localtime.Normalize(fs.SignedAt()), localtime.Normalize(ufs.SignedAt()))
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
