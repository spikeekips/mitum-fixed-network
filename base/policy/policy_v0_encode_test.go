package policy

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testPolicyV0Encode struct {
	suite.Suite

	priv key.Privatekey
	enc  encoder.Encoder
}

func (t *testPolicyV0Encode) SetupSuite() {
	t.priv = key.MustNewBTCPrivatekey()

	encs := encoder.NewEncoders()
	_ = encs.AddEncoder(t.enc)

	_ = encs.AddHinter(key.BTCPrivatekeyHinter)
	_ = encs.AddHinter(key.BTCPublickeyHinter)
	_ = encs.AddHinter(valuehash.SHA256{})
	_ = encs.AddHinter(PolicyV0{})
}

func (t *testPolicyV0Encode) TestEncode() {
	po := NewPolicyV0(3, 6, 9)
	t.NoError(po.IsValid(nil))

	b, err := t.enc.Marshal(po)
	t.NoError(err)

	hinter, err := t.enc.DecodeByHint(b)
	t.NoError(err)

	upo, ok := hinter.(PolicyV0)
	t.True(ok)

	t.NoError(upo.IsValid(nil))

	t.True(po.Hash().Equal(upo.Hash()))
	t.Equal(po.MaxOperationsInSeal(), upo.MaxOperationsInSeal())
	t.Equal(po.MaxOperationsInProposal(), upo.MaxOperationsInProposal())

	t.Equal(jsonenc.ToString(po), jsonenc.ToString(upo))
}

func TestPolicyV0EncodeJSON(t *testing.T) {
	b := new(testPolicyV0Encode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestPolicyV0EncodeBSON(t *testing.T) {
	b := new(testPolicyV0Encode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
