package ballot

import (
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testProposalV0Encode struct {
	suite.Suite

	pk  key.Privatekey
	enc encoder.Encoder
}

func (t *testProposalV0Encode) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()

	encs := encoder.NewEncoders()
	t.NoError(encs.AddEncoder(t.enc))
	t.NoError(encs.TestAddHinter(base.StringAddress("")))
	t.NoError(encs.TestAddHinter(key.BTCPublickeyHinter))
	t.NoError(encs.TestAddHinter(ProposalV0{}))
	t.NoError(encs.TestAddHinter(base.DummyVoteproof{}))
}

func (t *testProposalV0Encode) TestEncode() {
	vp := base.NewDummyVoteproof(
		base.Height(10),
		base.Round(0),
		base.StageINIT,
		base.VoteResultMajority,
	)

	ib := ProposalV0{
		BaseBallotV0: BaseBallotV0{
			node: base.RandomStringAddress(),
		},
		ProposalFactV0: ProposalFactV0{
			BaseFactV0: BaseFactV0{
				height: base.Height(10),
				round:  base.Round(0),
			},
			seals: []valuehash.Hash{
				valuehash.RandomSHA256(),
				valuehash.RandomSHA256(),
				valuehash.RandomSHA256(),
			},
		},
		voteproof: vp,
	}

	t.NoError(ib.Sign(t.pk, nil))

	b, err := t.enc.Marshal(ib)
	t.NoError(err)

	ht, err := t.enc.Decode(b)
	t.NoError(err)

	nib, ok := ht.(ProposalV0)
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
	t.Equal(ib.FactSignature(), nib.FactSignature())
	t.True(ib.Fact().Hash().Equal(nib.Fact().Hash()))
	t.Equal(vp, nib.Voteproof())

	for i, s := range ib.Seals() {
		t.True(s.Equal(nib.Seals()[i]))
	}
}

func TestProposalV0EncodeJSON(t *testing.T) {
	b := new(testProposalV0Encode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestProposalV0EncodeBSON(t *testing.T) {
	b := new(testProposalV0Encode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
