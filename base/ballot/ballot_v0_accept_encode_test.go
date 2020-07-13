package ballot

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testBallotV0ACCEPTEncode struct {
	suite.Suite

	pk  key.Privatekey
	enc encoder.Encoder
}

func (t *testBallotV0ACCEPTEncode) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()

	encs := encoder.NewEncoders()
	t.NoError(encs.AddEncoder(t.enc))
	t.NoError(encs.AddHinter(valuehash.SHA256{}))
	t.NoError(encs.AddHinter(base.NewShortAddress("")))
	t.NoError(encs.AddHinter(key.BTCPublickey{}))
	t.NoError(encs.AddHinter(ACCEPTBallotV0{}))
	t.NoError(encs.AddHinter(base.DummyVoteproof{}))
}

func (t *testBallotV0ACCEPTEncode) TestEncode() {
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

	b, err := t.enc.Marshal(ab)
	t.NoError(err)

	ht, err := t.enc.DecodeByHint(b)
	t.NoError(err)

	nib, ok := ht.(ACCEPTBallotV0)
	t.True(ok)

	t.NoError(nib.IsValid(nil))

	t.Equal(ab.Node(), nib.Node())
	t.Equal(ab.Signature(), nib.Signature())
	t.Equal(ab.Height(), nib.Height())
	t.Equal(ab.Round(), nib.Round())
	t.Equal(localtime.Normalize(ab.SignedAt()), localtime.Normalize(nib.SignedAt()))
	t.True(ab.Signer().Equal(nib.Signer()))
	t.True(ab.Hash().Equal(nib.Hash()))
	t.True(ab.BodyHash().Equal(nib.BodyHash()))
	t.True(ab.NewBlock().Equal(nib.NewBlock()))
	t.True(ab.Proposal().Equal(nib.Proposal()))
	t.Equal(ab.FactSignature(), nib.FactSignature())
	t.True(ab.Fact().Hash().Equal(nib.Fact().Hash()))
	t.Equal(vp, nib.Voteproof())
}

func TestBallotV0ACCEPTEncodeJSON(t *testing.T) {
	b := new(testBallotV0ACCEPTEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestBallotV0ACCEPTEncodeBSON(t *testing.T) {
	b := new(testBallotV0ACCEPTEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
