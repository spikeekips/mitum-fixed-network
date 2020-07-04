package ballot

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testBallotV0ACCEPTJSON struct {
	suite.Suite

	pk key.BTCPrivatekey
}

func (t *testBallotV0ACCEPTJSON) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testBallotV0ACCEPTJSON) TestEncode() {
	je := jsonenc.NewEncoder()

	encs := encoder.NewEncoders()
	t.NoError(encs.AddEncoder(je))
	t.NoError(encs.AddHinter(valuehash.SHA256{}))
	t.NoError(encs.AddHinter(base.NewShortAddress("")))
	t.NoError(encs.AddHinter(key.BTCPublickey{}))
	t.NoError(encs.AddHinter(ACCEPTBallotV0{}))
	t.NoError(encs.AddHinter(base.DummyVoteproof{}))

	vp := base.NewDummyVoteproof(
		base.Height(10),
		base.Round(0),
		base.StageINIT,
		base.VoteResultMajority,
	)

	ab := ACCEPTBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: base.NewShortAddress("test-for-accept-ballot"),
		},
		ACCEPTBallotFactV0: ACCEPTBallotFactV0{
			BaseBallotFactV0: BaseBallotFactV0{
				height: vp.Height(),
				round:  vp.Round(),
			},
			proposal: valuehash.RandomSHA256(),
			newBlock: valuehash.RandomSHA256(),
		},
		voteproof: vp,
	}

	t.NoError(ab.Sign(t.pk, nil))

	b, err := je.Marshal(ab)
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
	t.True(ab.Fact().Hash().Equal(nib.Fact().Hash()))
	t.Equal(vp, nib.Voteproof())
}

func TestBallotV0ACCEPTJSON(t *testing.T) {
	suite.Run(t, new(testBallotV0ACCEPTJSON))
}
