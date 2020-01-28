package mitum

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/valuehash"
)

type testBallotV0SIGN struct {
	suite.Suite

	pk key.BTCPrivatekey
}

func (t *testBallotV0SIGN) SetupSuite() {
	_ = hint.RegisterType(key.BTCPrivatekey{}.Hint().Type(), "btc-privatekey")
	_ = hint.RegisterType(key.BTCPublickey{}.Hint().Type(), "btc-publickey")
	_ = hint.RegisterType(SIGNBallotType, "sign-ballot")

	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testBallotV0SIGN) TestNew() {
	ib := SIGNBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: NewShortAddress("test-for-sign-ballot"),
		},
		SIGNBallotV0Fact: SIGNBallotV0Fact{
			BaseBallotV0Fact: BaseBallotV0Fact{
				height: Height(10),
				round:  Round(0),
			},
			proposal: valuehash.RandomSHA256(),
			newBlock: valuehash.RandomSHA256(),
		},
	}

	t.NotEmpty(ib)

	_ = (interface{})(ib).(Ballot)
	t.Implements((*Ballot)(nil), ib)
}

func (t *testBallotV0SIGN) TestGenerateHash() {
	ib := SIGNBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: NewShortAddress("test-for-sign-ballot"),
		},
		SIGNBallotV0Fact: SIGNBallotV0Fact{
			BaseBallotV0Fact: BaseBallotV0Fact{
				height: Height(10),
				round:  Round(0),
			},
			proposal: valuehash.RandomSHA256(),
			newBlock: valuehash.RandomSHA256(),
		},
	}

	h, err := ib.GenerateBodyHash(nil)
	t.NoError(err)
	t.NotNil(h)
	t.NotEmpty(h)

	bh, err := ib.GenerateBodyHash(nil)
	t.NoError(err)
	t.NotNil(bh)
	t.NotEmpty(bh)
}

func (t *testBallotV0SIGN) TestSign() {
	ib := SIGNBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: NewShortAddress("test-for-sign-ballot"),
		},
		SIGNBallotV0Fact: SIGNBallotV0Fact{
			BaseBallotV0Fact: BaseBallotV0Fact{
				height: Height(10),
				round:  Round(0),
			},
			proposal: valuehash.RandomSHA256(),
			newBlock: valuehash.RandomSHA256(),
		},
	}

	t.Nil(ib.Hash())
	t.Nil(ib.BodyHash())
	t.Nil(ib.Signer())
	t.Nil(ib.Signature())
	t.True(ib.SignedAt().IsZero())

	t.NoError(ib.Sign(t.pk, nil))

	t.NotNil(ib.Hash())
	t.NotNil(ib.BodyHash())
	t.NotNil(ib.Signer())
	t.NotNil(ib.Signature())
	t.False(ib.SignedAt().IsZero())

	t.NoError(ib.Signer().Verify(ib.BodyHash().Bytes(), ib.Signature()))

	// invalid signature
	unknownPK, _ := key.NewBTCPrivatekey()
	err := unknownPK.Publickey().Verify(ib.BodyHash().Bytes(), ib.Signature())
	t.True(xerrors.Is(err, key.SignatureVerificationFailedError))
}

func (t *testBallotV0SIGN) TestIsValid() {
	{ // empty signedAt
		bb := BaseBallotV0{
			node: NewShortAddress("test-for-sign-ballot"),
		}
		err := bb.IsValid(nil)
		t.Contains(err.Error(), "empty SignedAt")
	}

	{ // empty signer
		bb := BaseBallotV0{
			node:     NewShortAddress("test-for-sign-ballot"),
			signedAt: localtime.Now(),
		}
		err := bb.IsValid(nil)
		t.Contains(err.Error(), "empty Signer")
	}

	{ // empty signature
		bb := BaseBallotV0{
			node:     NewShortAddress("test-for-sign-ballot"),
			signedAt: localtime.Now(),
			signer:   t.pk.Publickey(),
		}
		err := bb.IsValid(nil)
		t.Contains(err.Error(), "empty Signature")
	}
}

func TestBallotV0SIGN(t *testing.T) {
	suite.Run(t, new(testBallotV0SIGN))
}
