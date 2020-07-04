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

type testBallotProposalV0Encode struct {
	suite.Suite

	pk  key.BTCPrivatekey
	enc encoder.Encoder
}

func (t *testBallotProposalV0Encode) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()

	encs := encoder.NewEncoders()
	t.NoError(encs.AddEncoder(t.enc))
	t.NoError(encs.AddHinter(valuehash.SHA256{}))
	t.NoError(encs.AddHinter(base.NewShortAddress("")))
	t.NoError(encs.AddHinter(key.BTCPublickey{}))
	t.NoError(encs.AddHinter(ProposalV0{}))
}

func (t *testBallotProposalV0Encode) TestEncode() {
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

	b, err := t.enc.Marshal(ib)
	t.NoError(err)

	ht, err := t.enc.DecodeByHint(b)
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

func TestBallotProposalV0EncodeJSON(t *testing.T) {
	b := new(testBallotProposalV0Encode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestBallotProposalV0EncodeBSON(t *testing.T) {
	b := new(testBallotProposalV0Encode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
