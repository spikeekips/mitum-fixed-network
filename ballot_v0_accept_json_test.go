package mitum

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/valuehash"
)

type testBallotV0ACCEPTJSON struct {
	suite.Suite

	pk key.BTCPrivatekey
}

func (t *testBallotV0ACCEPTJSON) SetupSuite() {
	_ = hint.RegisterType(key.BTCPrivatekey{}.Hint().Type(), "btc-privatekey")
	_ = hint.RegisterType(key.BTCPublickey{}.Hint().Type(), "btc-publickey")
	_ = hint.RegisterType(valuehash.SHA256{}.Hint().Type(), "sha256")
	_ = hint.RegisterType(encoder.JSONEncoder{}.Hint().Type(), "json-encoder")
	_ = hint.RegisterType((NewShortAddress("")).Hint().Type(), "short-address")
	_ = hint.RegisterType(ACCEPTBallotType, "accept-ballot")

	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testBallotV0ACCEPTJSON) TestEncode() {
	je := encoder.NewJSONEncoder()

	encs := encoder.NewEncoders()
	t.NoError(encs.AddEncoder(je))
	t.NoError(encs.AddHinter(valuehash.SHA256{}))
	t.NoError(encs.AddHinter(NewShortAddress("")))
	t.NoError(encs.AddHinter(key.BTCPublickey{}))
	t.NoError(encs.AddHinter(ACCEPTBallotV0{}))

	ab := ACCEPTBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: NewShortAddress("test-for-accept-ballot"),
		},
		ACCEPTBallotFactV0: ACCEPTBallotFactV0{
			BaseBallotFactV0: BaseBallotFactV0{
				height: Height(10),
				round:  Round(0),
			},
			proposal: valuehash.RandomSHA256(),
			newBlock: valuehash.RandomSHA256(),
		},
	}

	t.NoError(ab.Sign(t.pk, nil))

	b, err := je.Encode(ab)
	t.NoError(err)

	ht, err := je.DecodeByHint(b)
	t.NoError(err)

	nib, ok := ht.(ACCEPTBallotV0)
	t.True(ok)
	t.NoError(nib.IsValid(nil))

	t.Equal(ab.Node(), nib.Node())
	t.Equal(ab.Signature(), nib.Signature())
	t.Equal(ab.Height(), nib.Height())
	t.Equal(ab.Round(), nib.Round())
	t.Equal(localtime.RFC3339(ab.SignedAt()), localtime.RFC3339(nib.SignedAt()))
	t.True(ab.Signer().Equal(nib.Signer()))
	t.True(ab.Hash().Equal(nib.Hash()))
	t.True(ab.BodyHash().Equal(nib.BodyHash()))
	t.True(ab.NewBlock().Equal(nib.NewBlock()))
	t.True(ab.Proposal().Equal(nib.Proposal()))
	t.Equal(ab.FactSignature(), nib.FactSignature())
	t.True(ab.FactHash().Equal(nib.FactHash()))
}

func TestBallotV0ACCEPTJSON(t *testing.T) {
	suite.Run(t, new(testBallotV0ACCEPTJSON))
}
