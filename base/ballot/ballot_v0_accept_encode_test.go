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

type testACCEPTV0Encode struct {
	suite.Suite

	pk  key.Privatekey
	enc encoder.Encoder
}

func (t *testACCEPTV0Encode) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()

	encs := encoder.NewEncoders()
	t.NoError(encs.AddEncoder(t.enc))
	t.NoError(encs.TestAddHinter(base.StringAddress("")))
	t.NoError(encs.TestAddHinter(key.BTCPublickeyHinter))
	t.NoError(encs.TestAddHinter(ACCEPTV0{}))
	t.NoError(encs.TestAddHinter(base.DummyVoteproof{}))
}

func (t *testACCEPTV0Encode) TestEncode() {
	vp := base.NewDummyVoteproof(
		base.Height(10),
		base.Round(0),
		base.StageINIT,
		base.VoteResultMajority,
	)

	ab := ACCEPTV0{
		BaseBallotV0: BaseBallotV0{
			node: base.RandomStringAddress(),
		},
		ACCEPTFactV0: ACCEPTFactV0{
			BaseFactV0: BaseFactV0{
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

	nib, ok := ht.(ACCEPTV0)
	t.True(ok)

	t.NoError(nib.IsValid(nil))

	t.Equal(ab.Node(), nib.Node())
	t.Equal(ab.Signature(), nib.Signature())
	t.Equal(ab.Height(), nib.Height())
	t.Equal(ab.Round(), nib.Round())
	t.True(localtime.Equal(ab.SignedAt(), nib.SignedAt()))
	t.True(ab.Signer().Equal(nib.Signer()))
	t.True(ab.Hash().Equal(nib.Hash()))
	t.True(ab.BodyHash().Equal(nib.BodyHash()))
	t.True(ab.NewBlock().Equal(nib.NewBlock()))
	t.True(ab.Proposal().Equal(nib.Proposal()))
	t.Equal(ab.FactSignature(), nib.FactSignature())
	t.True(ab.Fact().Hash().Equal(nib.Fact().Hash()))
	t.Equal(vp, nib.Voteproof())
}

func TestACCEPTV0EncodeJSON(t *testing.T) {
	b := new(testACCEPTV0Encode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestACCEPTV0EncodeBSON(t *testing.T) {
	b := new(testACCEPTV0Encode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
