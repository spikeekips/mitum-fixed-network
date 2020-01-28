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

type testBallotProposalV0JSON struct {
	suite.Suite

	pk key.BTCPrivatekey
}

func (t *testBallotProposalV0JSON) SetupSuite() {
	_ = hint.RegisterType(key.BTCPrivatekey{}.Hint().Type(), "btc-privatekey")
	_ = hint.RegisterType(key.BTCPublickey{}.Hint().Type(), "btc-publickey")
	_ = hint.RegisterType(valuehash.SHA256{}.Hint().Type(), "sha256")
	_ = hint.RegisterType(encoder.JSONEncoder{}.Hint().Type(), "json-encoder")
	_ = hint.RegisterType((NewShortAddress("")).Hint().Type(), "short-address")
	_ = hint.RegisterType(ProposalBallotType, "proposal")

	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testBallotProposalV0JSON) TestEncode() {
	je := encoder.NewJSONEncoder()

	encs := encoder.NewEncoders()
	t.NoError(encs.AddEncoder(je))
	t.NoError(encs.AddHinter(valuehash.SHA256{}))
	t.NoError(encs.AddHinter(NewShortAddress("")))
	t.NoError(encs.AddHinter(key.BTCPublickey{}))
	t.NoError(encs.AddHinter(ProposalV0{}))

	ib := ProposalV0{
		BaseBallotV0: BaseBallotV0{
			node: NewShortAddress("test-for-proposal"),
		},
		ProposalV0Fact: ProposalV0Fact{
			BaseBallotV0Fact: BaseBallotV0Fact{
				height: Height(10),
				round:  Round(0),
			},
			seals: []valuehash.Hash{
				valuehash.RandomSHA256(),
				valuehash.RandomSHA256(),
				valuehash.RandomSHA256(),
			},
		},
	}

	t.NoError(ib.Sign(t.pk, nil))

	b, err := je.Encode(ib)
	t.NoError(err)

	ht, err := je.DecodeByHint(b)
	t.NoError(err)

	nib, ok := ht.(ProposalV0)
	t.True(ok)
	t.Equal(ib.Node(), nib.Node())
	t.Equal(ib.Signature(), nib.Signature())
	t.Equal(ib.Height(), nib.Height())
	t.Equal(ib.Round(), nib.Round())
	t.Equal(localtime.RFC3339(ib.SignedAt()), localtime.RFC3339(nib.SignedAt()))
	t.True(ib.Signer().Equal(nib.Signer()))
	t.True(ib.Hash().Equal(nib.Hash()))
	t.True(ib.BodyHash().Equal(nib.BodyHash()))

	for i, s := range ib.Seals() {
		t.True(s.Equal(nib.Seals()[i]))
	}
}

func TestBallotProposalV0JSON(t *testing.T) {
	suite.Run(t, new(testBallotProposalV0JSON))
}
