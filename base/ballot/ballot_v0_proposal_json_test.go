package ballot

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/localtime"
)

type testBallotProposalV0JSON struct {
	suite.Suite

	pk key.BTCPrivatekey
}

func (t *testBallotProposalV0JSON) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testBallotProposalV0JSON) TestEncode() {
	je := encoder.NewJSONEncoder()

	encs := encoder.NewEncoders()
	t.NoError(encs.AddEncoder(je))
	t.NoError(encs.AddHinter(valuehash.SHA256{}))
	t.NoError(encs.AddHinter(base.NewShortAddress("")))
	t.NoError(encs.AddHinter(key.BTCPublickey{}))
	t.NoError(encs.AddHinter(ProposalV0{}))

	ib := ProposalV0{
		BaseBallotV0: BaseBallotV0{
			node: base.NewShortAddress("test-for-proposal"),
		},
		ProposalFactV0: ProposalFactV0{
			BaseBallotFactV0: BaseBallotFactV0{
				height: base.Height(10),
				round:  base.Round(0),
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
	t.NoError(nib.IsValid(nil))
	t.Equal(ib.Node(), nib.Node())
	t.Equal(ib.Signature(), nib.Signature())
	t.Equal(ib.Height(), nib.Height())
	t.Equal(ib.Round(), nib.Round())
	t.Equal(localtime.RFC3339(ib.SignedAt()), localtime.RFC3339(nib.SignedAt()))
	t.True(ib.Signer().Equal(nib.Signer()))
	t.True(ib.Hash().Equal(nib.Hash()))
	t.True(ib.BodyHash().Equal(nib.BodyHash()))
	t.Equal(ib.FactSignature(), nib.FactSignature())
	t.True(ib.FactHash().Equal(nib.FactHash()))

	for i, s := range ib.Seals() {
		t.True(s.Equal(nib.Seals()[i]))
	}
}

func TestBallotProposalV0JSON(t *testing.T) {
	suite.Run(t, new(testBallotProposalV0JSON))
}
