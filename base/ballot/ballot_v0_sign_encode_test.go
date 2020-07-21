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

type testBallotV0SIGNEncode struct {
	suite.Suite

	pk  key.Privatekey
	enc encoder.Encoder
}

func (t *testBallotV0SIGNEncode) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()

	encs := encoder.NewEncoders()
	t.NoError(encs.AddEncoder(t.enc))
	t.NoError(encs.AddHinter(valuehash.SHA256{}))
	t.NoError(encs.AddHinter(base.StringAddress("")))
	t.NoError(encs.AddHinter(key.BTCPublickeyHinter))
	t.NoError(encs.AddHinter(SIGNBallotV0{}))
}

func (t *testBallotV0SIGNEncode) TestEncode() {
	ib := SIGNBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: base.RandomStringAddress(),
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

	b, err := t.enc.Marshal(ib)
	t.NoError(err)

	ht, err := t.enc.DecodeByHint(b)
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
	t.True(ib.Fact().Hash().Equal(nib.Fact().Hash()))
}

func TestBallotV0SIGNEncodeJSON(t *testing.T) {
	b := new(testBallotV0SIGNEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestBallotV0SIGNEncodeBSON(t *testing.T) {
	b := new(testBallotV0SIGNEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
