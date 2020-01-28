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

type testBallotV0SIGNJSON struct {
	suite.Suite

	pk key.BTCPrivatekey
}

func (t *testBallotV0SIGNJSON) SetupSuite() {
	_ = hint.RegisterType(key.BTCPrivatekey{}.Hint().Type(), "btc-privatekey")
	_ = hint.RegisterType(key.BTCPublickey{}.Hint().Type(), "btc-publickey")
	_ = hint.RegisterType(valuehash.SHA256{}.Hint().Type(), "sha256")
	_ = hint.RegisterType(encoder.JSONEncoder{}.Hint().Type(), "json-encoder")
	_ = hint.RegisterType((NewShortAddress("")).Hint().Type(), "short-address")
	_ = hint.RegisterType(SIGNBallotType, "sign-ballot")

	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testBallotV0SIGNJSON) TestEncode() {
	je := encoder.NewJSONEncoder()

	encs := encoder.NewEncoders()
	t.NoError(encs.AddEncoder(je))
	t.NoError(encs.AddHinter(valuehash.SHA256{}))
	t.NoError(encs.AddHinter(NewShortAddress("")))
	t.NoError(encs.AddHinter(key.BTCPublickey{}))
	t.NoError(encs.AddHinter(SIGNBallotV0{}))

	ib := SIGNBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: NewShortAddress("test-for-sign-ballot"),
		},
		SIGNBallotV0Fact: SIGNBallotV0Fact{
			BaseBallotV0Fact: BaseBallotV0Fact{
				height: Height(10),
				round:  Round(0),
			},
			proposal: valuehash.RandomSHA256(),
			newBlock: valuehash.RandomSHA256(),
		},
	}

	t.NoError(ib.Sign(t.pk, nil))

	b, err := je.Encode(ib)
	t.NoError(err)

	ht, err := je.DecodeByHint(b)
	t.NoError(err)

	nib, ok := ht.(SIGNBallotV0)
	t.True(ok)
	t.Equal(ib.Node(), nib.Node())
	t.Equal(ib.Signature(), nib.Signature())
	t.Equal(ib.Height(), nib.Height())
	t.Equal(ib.Round(), nib.Round())
	t.Equal(localtime.RFC3339(ib.SignedAt()), localtime.RFC3339(nib.SignedAt()))
	t.True(ib.Signer().Equal(nib.Signer()))
	t.True(ib.Hash().Equal(nib.Hash()))
	t.True(ib.BodyHash().Equal(nib.BodyHash()))
	t.True(ib.NewBlock().Equal(nib.NewBlock()))
	t.True(ib.Proposal().Equal(nib.Proposal()))
}

func TestBallotV0SIGNJSON(t *testing.T) {
	suite.Run(t, new(testBallotV0SIGNJSON))
}
