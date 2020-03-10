package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/operation"
	"github.com/spikeekips/mitum/valuehash"
)

type testBallotV0ACCEPT struct {
	suite.Suite

	pk key.BTCPrivatekey
}

func (t *testBallotV0ACCEPT) SetupSuite() {
	_ = hint.RegisterType(key.BTCPrivatekey{}.Hint().Type(), "btc-privatekey")
	_ = hint.RegisterType(key.BTCPublickey{}.Hint().Type(), "btc-publickey")
	_ = hint.RegisterType(ACCEPTBallotType, "accept-ballot")

	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testBallotV0ACCEPT) TestNew() {
	vp := NewDummyVoteproof(
		Height(10),
		Round(0),
		StageINIT,
		VoteproofMajority,
	)

	ib := ACCEPTBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: NewShortAddress("test-for-accept-ballot"),
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

	t.NotEmpty(ib)

	_ = (interface{})(ib).(Ballot)
	t.Implements((*Ballot)(nil), ib)
}

func (t *testBallotV0ACCEPT) TestFact() {
	vp := NewDummyVoteproof(
		Height(10),
		Round(0),
		StageINIT,
		VoteproofMajority,
	)

	ib := ACCEPTBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: NewShortAddress("test-for-accept-ballot"),
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

	t.Implements((*operation.FactSeal)(nil), ib)

	fact := ib.Fact()

	_ = (interface{})(fact).(operation.Fact)

	factHash := fact.Hash()
	t.NotNil(factHash)
	t.NoError(fact.IsValid(nil))

	// before signing, FactHash() and FactSignature() is nil
	t.Nil(ib.FactHash())
	t.Nil(ib.FactSignature())

	t.NoError(ib.Sign(t.pk, nil))

	t.NotNil(ib.FactHash())
	t.NotNil(ib.FactSignature())

	t.NoError(ib.Signer().Verify(ib.FactHash().Bytes(), ib.FactSignature()))
}

func (t *testBallotV0ACCEPT) TestGenerateHash() {
	vp := NewDummyVoteproof(
		Height(10),
		Round(0),
		StageINIT,
		VoteproofMajority,
	)

	ib := ACCEPTBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: NewShortAddress("test-for-accept-ballot"),
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

	h, err := ib.GenerateBodyHash()
	t.NoError(err)
	t.NotNil(h)
	t.NotEmpty(h)

	bh, err := ib.GenerateBodyHash()
	t.NoError(err)
	t.NotNil(bh)
	t.NotEmpty(bh)
}

func (t *testBallotV0ACCEPT) TestSign() {
	vp := NewDummyVoteproof(
		Height(10),
		Round(0),
		StageINIT,
		VoteproofMajority,
	)

	ib := ACCEPTBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: NewShortAddress("test-for-accept-ballot"),
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

func (t *testBallotV0ACCEPT) TestIsValid() {
	{ // empty signedAt
		bb := BaseBallotV0{
			node: NewShortAddress("test-for-accept-ballot"),
		}
		err := bb.IsValid(nil)
		t.Contains(err.Error(), "empty SignedAt")
	}

	{ // empty signer
		bb := BaseBallotV0{
			node:     NewShortAddress("test-for-accept-ballot"),
			signedAt: localtime.Now(),
		}
		err := bb.IsValid(nil)
		t.Contains(err.Error(), "empty Signer")
	}

	{ // empty signature
		bb := BaseBallotV0{
			node:     NewShortAddress("test-for-accept-ballot"),
			signedAt: localtime.Now(),
			signer:   t.pk.Publickey(),
		}
		err := bb.IsValid(nil)
		t.Contains(err.Error(), "empty Signature")
	}
}

func TestBallotV0ACCEPT(t *testing.T) {
	suite.Run(t, new(testBallotV0ACCEPT))
}
