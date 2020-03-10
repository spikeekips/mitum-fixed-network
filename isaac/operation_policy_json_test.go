package isaac

import (
	"testing"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
	"github.com/stretchr/testify/suite"
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
	t.enc = encoder.NewJSONEncoder()
	_ = t.encs.AddEncoder(t.enc)

	_ = t.encs.AddHinter(key.BTCPrivatekey{})
	_ = t.encs.AddHinter(key.BTCPublickey{})
	_ = t.encs.AddHinter(valuehash.SHA256{})
	_ = t.encs.AddHinter(SetPolicyOperationFactV0{})
	_ = t.encs.AddHinter(SetPolicyOperationV0{})
}

func (t *testSetPolicyOperationJSON) TestEncode() {
	token := []byte("findme")
	spo, err := NewSetPolicyOperationV0(t.pk, token, nil)
	t.NoError(err)

	spo.NumberOfActingSuffrageNodes = 1

	threshold, err := NewThreshold(3, 99.99)
	t.NoError(err)
	spo.Threshold = threshold

	t.NoError(spo.IsValid(nil))

	b, err := util.JSONMarshal(spo)
	t.NoError(err)

	hinter, err := t.enc.DecodeByHint(b)
	t.NoError(err)

	uspo, ok := hinter.(SetPolicyOperationV0)
	t.True(ok)

	t.NoError(uspo.IsValid(nil))

	t.True(spo.Hash().Equal(uspo.Hash()))
	t.Equal(spo.Threshold, uspo.Threshold)

	t.Equal(util.ToString(spo), util.ToString(uspo))
}

func TestSetPolicyOperationJSON(t *testing.T) {
	suite.Run(t, new(testSetPolicyOperationJSON))
}
