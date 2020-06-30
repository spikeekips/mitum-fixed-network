package ballot

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testBallotV0Proposal struct {
	suite.Suite

	pk key.BTCPrivatekey
}

func (t *testBallotV0Proposal) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testBallotV0Proposal) TestNew() {
	ib := ProposalV0{
		BaseBallotV0: BaseBallotV0{
			node: base.NewShortAddress("test-for-proposal"),
		},
		ProposalFactV0: ProposalFactV0{
			BaseBallotFactV0: BaseBallotFactV0{
				height: base.Height(10),
				round:  base.Round(0),
			},
		},
	}

	t.NotEmpty(ib)

	_ = (interface{})(ib).(Ballot)
	t.Implements((*Ballot)(nil), ib)
}

func (t *testBallotV0Proposal) TestFact() {
	ib := ProposalV0{
		BaseBallotV0: BaseBallotV0{
			node: base.NewShortAddress("test-for-proposal"),
		},
		ProposalFactV0: ProposalFactV0{
			BaseBallotFactV0: BaseBallotFactV0{
				height: base.Height(10),
				round:  base.Round(0),
			},
		},
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

func (t *testBallotV0Proposal) TestGenerateHash() {
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

	h, err := ib.GenerateBodyHash()
	t.NoError(err)
	t.NotNil(h)
	t.NotEmpty(h)

	bh, err := ib.GenerateBodyHash()
	t.NoError(err)
	t.NotNil(bh)
	t.NotEmpty(bh)
}

func (t *testBallotV0Proposal) TestSign() {
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

func TestBallotV0Proposal(t *testing.T) {
	suite.Run(t, new(testBallotV0Proposal))
}
