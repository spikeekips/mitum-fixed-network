package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testSetPolicyOperationJSON struct {
	suite.Suite

	pk   key.BTCPrivatekey
	encs *encoder.Encoders
	enc  encoder.Encoder
}

func (t *testSetPolicyOperationJSON) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()

	t.encs = encoder.NewEncoders()
	t.enc = jsonenc.NewEncoder()
	_ = t.encs.AddEncoder(t.enc)

	_ = t.encs.AddHinter(key.BTCPrivatekey{})
	_ = t.encs.AddHinter(key.BTCPublickey{})
	_ = t.encs.AddHinter(valuehash.SHA256{})
	_ = t.encs.AddHinter(SetPolicyOperationFactV0{})
	_ = t.encs.AddHinter(SetPolicyOperationV0{})
}

func (t *testSetPolicyOperationJSON) TestEncode() {
	token := []byte("findme")

	policies := DefaultPolicy()
	policies.thresholdRatio = base.ThresholdRatio(99.99)
	policies.numberOfActingSuffrageNodes = 1

	spo, err := NewSetPolicyOperationV0(t.pk, token, policies, nil)
	t.NoError(err)

	b, err := jsonenc.Marshal(spo)
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

func TestSetPolicyOperationJSON(t *testing.T) {
	suite.Run(t, new(testSetPolicyOperationJSON))
}
