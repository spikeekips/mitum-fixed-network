package ballot

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testINITV0 struct {
	suite.Suite

	pk key.Privatekey
}

func (t *testINITV0) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testINITV0) TestNew() {
	ib := INITV0{
		BaseBallotV0: BaseBallotV0{
			node: base.RandomStringAddress(),
		},
		INITFactV0: INITFactV0{
			BaseFactV0: BaseFactV0{
				height: base.Height(10),
				round:  base.Round(0),
			},
			previousBlock: valuehash.RandomSHA256(),
		},
	}

	t.NotEmpty(ib)

	_ = (interface{})(ib).(Ballot)
	t.Implements((*Ballot)(nil), ib)
}

func (t *testINITV0) TestFact() {
	vp := base.NewDummyVoteproof(
		base.Height(10),
		base.Round(0),
		base.StageACCEPT,
		base.VoteResultMajority,
	)

	ib := INITV0{
		BaseBallotV0: BaseBallotV0{
			node: base.RandomStringAddress(),
		},
		INITFactV0: INITFactV0{
			BaseFactV0: BaseFactV0{
				height: vp.Height() + 1,
				round:  base.Round(0),
			},
			previousBlock: valuehash.RandomSHA256(),
		},
		voteproof: vp,
	}

	fact := ib.Fact()

	_ = (interface{})(fact).(base.Fact)

	factHash := fact.Hash()
	t.NotNil(factHash)
	t.NoError(fact.IsValid(nil))

	t.Nil(ib.FactSignature())

	t.NoError(ib.Sign(t.pk, nil))

	t.NotNil(ib.Fact().Hash())
	t.NotNil(ib.FactSignature())

	t.NoError(ib.Signer().Verify(ib.Fact().Hash().Bytes(), ib.FactSignature()))
}

func (t *testINITV0) TestGenerateHash() {
	vp := base.NewDummyVoteproof(
		base.Height(10),
		base.Round(0),
		base.StageACCEPT,
		base.VoteResultMajority,
	)

	ib := INITV0{
		BaseBallotV0: BaseBallotV0{
			node: base.RandomStringAddress(),
		},
		INITFactV0: INITFactV0{
			BaseFactV0: BaseFactV0{
				height: vp.Height() + 1,
				round:  base.Round(0),
			},
			previousBlock: valuehash.RandomSHA256(),
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

func (t *testINITV0) TestSign() {
	vp := base.NewDummyVoteproof(
		base.Height(10),
		base.Round(0),
		base.StageACCEPT,
		base.VoteResultMajority,
	)

	ib := INITV0{
		BaseBallotV0: BaseBallotV0{
			node: base.RandomStringAddress(),
		},
		INITFactV0: INITFactV0{
			BaseFactV0: BaseFactV0{
				height: vp.Height() + 1,
				round:  base.Round(0),
			},
			previousBlock: valuehash.RandomSHA256(),
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

func (t *testINITV0) TestIsValid() {
	{ // empty signedAt
		bb := BaseBallotV0{
			node: base.RandomStringAddress(),
		}
		err := bb.IsValid(nil)
		t.Contains(err.Error(), "empty SignedAt")
	}

	{ // empty signer
		bb := BaseBallotV0{
			node:     base.RandomStringAddress(),
			signedAt: localtime.UTCNow(),
		}
		err := bb.IsValid(nil)
		t.Contains(err.Error(), "empty Signer")
	}

	{ // empty signature
		bb := BaseBallotV0{
			node:     base.RandomStringAddress(),
			signedAt: localtime.UTCNow(),
			signer:   t.pk.Publickey(),
		}
		err := bb.IsValid(nil)
		t.Contains(err.Error(), "empty Signature")
	}
}

func TestINITV0(t *testing.T) {
	suite.Run(t, new(testINITV0))
}
