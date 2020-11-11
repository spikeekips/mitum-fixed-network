package policy

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testSetPolicyV0Encode struct {
	suite.Suite

	priv key.Privatekey
	enc  encoder.Encoder
}

func (t *testSetPolicyV0Encode) SetupSuite() {
	t.priv = key.MustNewBTCPrivatekey()

	encs := encoder.NewEncoders()
	_ = encs.AddEncoder(t.enc)

	_ = encs.AddHinter(key.BTCPrivatekeyHinter)
	_ = encs.AddHinter(key.BTCPublickeyHinter)
	_ = encs.AddHinter(valuehash.SHA256{})
	_ = encs.AddHinter(operation.BaseFactSign{})
	_ = encs.AddHinter(PolicyV0{})
	_ = encs.AddHinter(SetPolicyV0{})
	_ = encs.AddHinter(SetPolicyFactV0{})
}

func (t *testSetPolicyV0Encode) TestEncode() {
	po := NewPolicyV0(3, 6, 9)
	networkID := util.UUID().Bytes()

	spo, err := NewSetPolicyV0(po, util.UUID().Bytes(), t.priv, networkID)
	t.NoError(err)
	t.NoError(spo.IsValid(networkID))

	b, err := t.enc.Marshal(spo)
	t.NoError(err)

	hinter, err := t.enc.DecodeByHint(b)
	t.NoError(err)

	uspo, ok := hinter.(SetPolicyV0)
	t.True(ok)
	t.NoError(uspo.IsValid(networkID))

	t.Equal(spo.Token(), uspo.Token())

	upo := uspo.SetPolicyFactV0.PolicyV0
	t.NoError(upo.IsValid(nil))

	t.True(po.Hash().Equal(upo.Hash()))
	t.Equal(po.MaxOperationsInSeal(), upo.MaxOperationsInSeal())
	t.Equal(po.MaxOperationsInProposal(), upo.MaxOperationsInProposal())

	t.Equal(jsonenc.ToString(po), jsonenc.ToString(upo))
}

func TestSetPolicyV0EncodeJSON(t *testing.T) {
	b := new(testSetPolicyV0Encode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestSetPolicyV0EncodeBSON(t *testing.T) {
	b := new(testSetPolicyV0Encode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
