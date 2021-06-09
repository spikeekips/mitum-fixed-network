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

type testBallotV0INITEncode struct {
	suite.Suite

	pk  key.Privatekey
	enc encoder.Encoder
}

func (t *testBallotV0INITEncode) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()

	encs := encoder.NewEncoders()
	t.NoError(encs.AddEncoder(t.enc))
	t.NoError(encs.TestAddHinter(base.StringAddress("")))
	t.NoError(encs.TestAddHinter(key.BTCPublickeyHinter))
	t.NoError(encs.TestAddHinter(INITBallotV0{}))
	t.NoError(encs.TestAddHinter(base.DummyVoteproof{}))
}

func (t *testBallotV0INITEncode) TestEncode() {
	vp := base.NewDummyVoteproof(
		base.Height(10),
		base.Round(0),
		base.StageACCEPT,
		base.VoteResultMajority,
	)

	ib := INITBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: base.RandomStringAddress(),
		},
		INITBallotFactV0: INITBallotFactV0{
			BaseBallotFactV0: BaseBallotFactV0{
				height: vp.Height() + 1,
				round:  base.Round(0),
			},
			previousBlock: valuehash.RandomSHA256(),
		},
		voteproof:       vp,
		acceptVoteproof: vp,
	}

	t.NoError(ib.Sign(t.pk, nil))

	b, err := t.enc.Marshal(ib)
	t.NoError(err)

	ht, err := t.enc.DecodeByHint(b)
	t.NoError(err)

	nib, ok := ht.(INITBallotV0)
	t.True(ok)

	t.NoError(nib.IsValid(nil))
	t.Equal(ib.Node(), nib.Node())
	t.Equal(ib.Signature(), nib.Signature())
	t.Equal(ib.Height(), nib.Height())
	t.Equal(ib.Round(), nib.Round())
	t.True(localtime.Equal(ib.SignedAt(), nib.SignedAt()))
	t.True(ib.Signer().Equal(nib.Signer()))
	t.True(ib.Hash().Equal(nib.Hash()))
	t.True(ib.BodyHash().Equal(nib.BodyHash()))
	t.True(ib.PreviousBlock().Equal(nib.PreviousBlock()))
	t.Equal(ib.FactSignature(), nib.FactSignature())
	t.True(ib.Fact().Hash().Equal(nib.Fact().Hash()))
	t.Equal(vp, nib.Voteproof())
}

func TestBallotV0INITEncodeJSON(t *testing.T) {
	b := new(testBallotV0INITEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestBallotV0INITEncodeBSON(t *testing.T) {
	b := new(testBallotV0INITEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
