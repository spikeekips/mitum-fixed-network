package ballot

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testProposalV0 struct {
	suite.Suite

	pk key.Privatekey
}

func (t *testProposalV0) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testProposalV0) TestNew() {
	ib := ProposalV0{
		BaseBallotV0: BaseBallotV0{
			node: base.RandomStringAddress(),
		},
		ProposalFactV0: ProposalFactV0{
			BaseFactV0: BaseFactV0{
				height: base.Height(10),
				round:  base.Round(0),
			},
		},
	}

	t.NotEmpty(ib)

	_ = (interface{})(ib).(Ballot)
	t.Implements((*Ballot)(nil), ib)
}

func (t *testProposalV0) TestFact() {
	ib := ProposalV0{
		BaseBallotV0: BaseBallotV0{
			node: base.RandomStringAddress(),
		},
		ProposalFactV0: ProposalFactV0{
			BaseFactV0: BaseFactV0{
				height: base.Height(10),
				round:  base.Round(0),
			},
		},
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

func (t *testProposalV0) TestGenerateHash() {
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

func (t *testProposalV0) TestSign() {
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

func TestProposalV0(t *testing.T) {
	suite.Run(t, new(testProposalV0))
}
