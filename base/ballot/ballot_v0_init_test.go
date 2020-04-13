package ballot

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/localtime"
)

type testBallotV0INIT struct {
	suite.Suite

	pk key.BTCPrivatekey
}

func (t *testBallotV0INIT) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testBallotV0INIT) TestNew() {
	ib := INITBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: base.NewShortAddress("test-for-init-ballot"),
		},
		INITBallotFactV0: INITBallotFactV0{
			BaseBallotFactV0: BaseBallotFactV0{
				height: base.Height(10),
				round:  base.Round(0),
			},
			previousBlock: valuehash.RandomSHA256(),
			previousRound: base.Round(0),
		},
	}

	t.NotEmpty(ib)

	_ = (interface{})(ib).(Ballot)
	t.Implements((*Ballot)(nil), ib)
}

func (t *testBallotV0INIT) TestFact() {
	vp := base.NewDummyVoteproof(
		base.Height(10),
		base.Round(0),
		base.StageACCEPT,
		base.VoteResultMajority,
	)

	ib := INITBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: base.NewShortAddress("test-for-init-ballot"),
		},
		INITBallotFactV0: INITBallotFactV0{
			BaseBallotFactV0: BaseBallotFactV0{
				height: vp.Height() + 1,
				round:  base.Round(0),
			},
			previousBlock: valuehash.RandomSHA256(),
			previousRound: vp.Round(),
		},
		voteproof: vp,
	}
	t.Implements((*operation.FactSeal)(nil), ib)

	fact := ib.Fact()

	_ = (interface{})(fact).(base.Fact)

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

func (t *testBallotV0INIT) TestGenerateHash() {
	vp := base.NewDummyVoteproof(
		base.Height(10),
		base.Round(0),
		base.StageACCEPT,
		base.VoteResultMajority,
	)

	ib := INITBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: base.NewShortAddress("test-for-init-ballot"),
		},
		INITBallotFactV0: INITBallotFactV0{
			BaseBallotFactV0: BaseBallotFactV0{
				height: vp.Height() + 1,
				round:  base.Round(0),
			},
			previousBlock: valuehash.RandomSHA256(),
			previousRound: base.Round(0),
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

func (t *testBallotV0INIT) TestSign() {
	vp := base.NewDummyVoteproof(
		base.Height(10),
		base.Round(0),
		base.StageACCEPT,
		base.VoteResultMajority,
	)

	ib := INITBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: base.NewShortAddress("test-for-init-ballot"),
		},
		INITBallotFactV0: INITBallotFactV0{
			BaseBallotFactV0: BaseBallotFactV0{
				height: vp.Height() + 1,
				round:  base.Round(0),
			},
			previousBlock: valuehash.RandomSHA256(),
			previousRound: base.Round(0),
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

func (t *testBallotV0INIT) TestIsValid() {
	{ // empty signedAt
		bb := BaseBallotV0{
			node: base.NewShortAddress("test-for-init-ballot"),
		}
		err := bb.IsValid(nil)
		t.Contains(err.Error(), "empty SignedAt")
	}

	{ // empty signer
		bb := BaseBallotV0{
			node:     base.NewShortAddress("test-for-init-ballot"),
			signedAt: localtime.Now(),
		}
		err := bb.IsValid(nil)
		t.Contains(err.Error(), "empty Signer")
	}

	{ // empty signature
		bb := BaseBallotV0{
			node:     base.NewShortAddress("test-for-init-ballot"),
			signedAt: localtime.Now(),
			signer:   t.pk.Publickey(),
		}
		err := bb.IsValid(nil)
		t.Contains(err.Error(), "empty Signature")
	}
}

func TestBallotV0INIT(t *testing.T) {
	suite.Run(t, new(testBallotV0INIT))
}
