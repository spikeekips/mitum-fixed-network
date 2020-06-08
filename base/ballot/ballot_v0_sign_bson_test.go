package ballot

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/localtime"
)

type testBallotV0SIGNBSON struct {
	suite.Suite

	pk key.BTCPrivatekey
}

func (t *testBallotV0SIGNBSON) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testBallotV0SIGNBSON) TestEncode() {
	je := bsonenc.NewEncoder()

	encs := encoder.NewEncoders()
	t.NoError(encs.AddEncoder(je))
	t.NoError(encs.AddHinter(valuehash.SHA256{}))
	t.NoError(encs.AddHinter(base.NewShortAddress("")))
	t.NoError(encs.AddHinter(key.BTCPublickey{}))
	t.NoError(encs.AddHinter(SIGNBallotV0{}))

	ib := SIGNBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: base.NewShortAddress("test-for-sign-ballot"),
		},
		SIGNBallotFactV0: SIGNBallotFactV0{
			BaseBallotFactV0: BaseBallotFactV0{
				height: base.Height(10),
				round:  base.Round(0),
			},
			proposal: valuehash.RandomSHA256(),
			newBlock: valuehash.RandomSHA256(),
		},
	}

	t.NoError(ib.Sign(t.pk, nil))

	b, err := je.Marshal(ib)
	t.NoError(err)

	ht, err := je.DecodeByHint(b)
	t.NoError(err)

	nib, ok := ht.(SIGNBallotV0)
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
	t.True(ib.NewBlock().Equal(nib.NewBlock()))
	t.True(ib.Proposal().Equal(nib.Proposal()))
	t.Equal(ib.FactSignature(), nib.FactSignature())
	t.True(ib.FactHash().Equal(nib.FactHash()))
}

func TestBallotV0SIGNBSON(t *testing.T) {
	suite.Run(t, new(testBallotV0SIGNBSON))
}
