package ballot

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/encoder"
	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/localtime"
)

type testBallotV0INITBSON struct {
	suite.Suite

	pk key.BTCPrivatekey
}

func (t *testBallotV0INITBSON) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testBallotV0INITBSON) TestEncode() {
	je := bsonencoder.NewEncoder()

	encs := encoder.NewEncoders()
	t.NoError(encs.AddEncoder(je))
	t.NoError(encs.AddHinter(valuehash.SHA256{}))
	t.NoError(encs.AddHinter(base.NewShortAddress("")))
	t.NoError(encs.AddHinter(key.BTCPublickey{}))
	t.NoError(encs.AddHinter(INITBallotV0{}))
	t.NoError(encs.AddHinter(base.DummyVoteproof{}))

	vp := base.NewDummyVoteproof(
		base.Height(10),
		base.Round(0),
		base.StageACCEPT,
		base.VoteResultMajority,
	)

	ib := INITBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: base.NewShortAddress("test-for-init-ballot"),
		},
		INITBallotFactV0: INITBallotFactV0{
			BaseBallotFactV0: BaseBallotFactV0{
				height: vp.Height() + 1,
				round:  base.Round(0),
			},
			previousBlock: valuehash.RandomSHA256(),
		},
		voteproof: vp,
	}

	t.NoError(ib.Sign(t.pk, nil))

	b, err := je.Marshal(ib)
	t.NoError(err)

	ht, err := je.DecodeByHint(b)
	t.NoError(err)

	nib, ok := ht.(INITBallotV0)
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
	t.True(ib.PreviousBlock().Equal(nib.PreviousBlock()))
	t.Equal(ib.FactSignature(), nib.FactSignature())
	t.True(ib.FactHash().Equal(nib.FactHash()))
	t.Equal(vp, nib.Voteproof())
}

func TestBallotV0INITBSON(t *testing.T) {
	suite.Run(t, new(testBallotV0INITBSON))
}
