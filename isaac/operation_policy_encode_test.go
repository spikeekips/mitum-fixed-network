package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testSetPolicyOperationEncode struct {
	suite.Suite

	pk  key.Privatekey
	enc encoder.Encoder
}

func (t *testSetPolicyOperationEncode) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()

	encs := encoder.NewEncoders()
	_ = encs.AddEncoder(t.enc)

	_ = encs.AddHinter(key.BTCPrivatekey{})
	_ = encs.AddHinter(key.BTCPublickey{})
	_ = encs.AddHinter(valuehash.SHA256{})
	_ = encs.AddHinter(SetPolicyOperationFactV0{})
	_ = encs.AddHinter(SetPolicyOperationV0{})
	_ = encs.AddHinter(operation.BaseFactSign{})
}

func (t *testSetPolicyOperationEncode) TestEncode() {
	token := []byte("findme")

	policies := DefaultPolicy()
	policies.thresholdRatio = base.ThresholdRatio(99.99)
	policies.numberOfActingSuffrageNodes = 1

	spo, err := NewSetPolicyOperationV0(t.pk, token, policies, nil)
	t.NoError(err)
	t.NoError(spo.IsValid(nil))

	b, err := t.enc.Marshal(spo)
	t.NoError(err)

	hinter, err := t.enc.DecodeByHint(b)
	t.NoError(err)

	uspo, ok := hinter.(SetPolicyOperationV0)
	t.True(ok)

	t.NoError(uspo.IsValid(nil))

	t.True(spo.Hash().Equal(uspo.Hash()))
	t.Equal(spo.ThresholdRatio(), uspo.ThresholdRatio())

	t.Equal(jsonenc.ToString(spo), jsonenc.ToString(uspo))
}

func TestSetPolicyOperationEncodeJSON(t *testing.T) {
	b := new(testSetPolicyOperationEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestSetPolicyOperationEncodeBSON(t *testing.T) {
	b := new(testSetPolicyOperationEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
