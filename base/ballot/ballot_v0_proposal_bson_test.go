package ballot

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testBallotProposalV0BSON struct {
	suite.Suite

	pk key.BTCPrivatekey
}

func (t *testBallotProposalV0BSON) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testBallotProposalV0BSON) TestEncode() {
	je := bsonenc.NewEncoder()

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

	b, err := je.Marshal(ib)
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
	t.Equal(localtime.Normalize(ib.SignedAt()), localtime.Normalize(nib.SignedAt()))
	t.True(ib.Signer().Equal(nib.Signer()))
	t.True(ib.Hash().Equal(nib.Hash()))
	t.True(ib.BodyHash().Equal(nib.BodyHash()))
	t.Equal(ib.FactSignature(), nib.FactSignature())
	t.True(ib.Fact().Hash().Equal(nib.Fact().Hash()))

	for i, s := range ib.Seals() {
		t.True(s.Equal(nib.Seals()[i]))
	}
}

func TestBallotProposalV0BSON(t *testing.T) {
	suite.Run(t, new(testBallotProposalV0BSON))
}
