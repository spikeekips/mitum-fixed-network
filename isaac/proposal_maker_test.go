package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testProposalMaker struct {
	baseTestStateHandler
}

func (t *testProposalMaker) TestCached() {
	proposalMaker := NewProposalMaker(t.locals(1)[0])

	round := base.Round(1)
	proposal, err := proposalMaker.Proposal(round)
	t.NoError(err)

	newProposal, err := proposalMaker.Proposal(round)
	t.NoError(err)

	t.True(proposal.Hash().Equal(newProposal.Hash()))
}

func (t *testProposalMaker) TestClean() {
	local := t.locals(1)[0]

	proposalMaker := NewProposalMaker(local)

	round := base.Round(1)
	_, err := proposalMaker.Proposal(round)
	t.NoError(err)

	t.NotNil(proposalMaker.proposed)
}

func (t *testProposalMaker) TestSeals() {
	local := t.locals(1)[0]

	var seals []seal.Seal
	// 10 operation.Seal
	for i := 0; i < 10; i++ {
		sl := t.newOperationSeal(local, 1)

		seals = append(seals, sl)
	}
	t.NoError(local.Storage().NewSeals(seals))

	proposalMaker := NewProposalMaker(local)

	round := base.Round(1)
	proposal, err := proposalMaker.Proposal(round)
	t.NoError(err)

	t.Equal(len(seals), len(proposal.Seals()))

	var expectedSeals []valuehash.Hash
	err = local.Storage().StagedOperationSeals(func(sl operation.Seal) (bool, error) {
		expectedSeals = append(expectedSeals, sl.Hash())

		return true, nil
	},
		true,
	)
	t.NoError(err)

	for i, h := range proposal.Seals() {
		t.True(expectedSeals[i].Equal(h))
	}
}

func (t *testProposalMaker) TestOneSealOver0() {
	local := t.locals(1)[0]

	var maxOperations uint = 3
	_, _ = local.Policy().SetMaxOperationsInProposal(maxOperations)

	var seals []seal.Seal
	for i := 0; i < int(maxOperations-1); i++ {
		sl := t.newOperationSeal(local, 1)
		seals = append(seals, sl)
	}

	over := t.newOperationSeal(local, 2)
	seals = append(seals, over)

	t.NoError(local.Storage().NewSeals(seals))

	proposalMaker := NewProposalMaker(local)

	round := base.Round(1)
	proposal, err := proposalMaker.Proposal(round)
	t.NoError(err)

	t.Equal(len(seals)-1, len(proposal.Seals()))

	for _, h := range proposal.Seals() {
		t.False(over.Hash().Equal(h))
	}
}

func (t *testProposalMaker) TestOneSealOver1() {
	local := t.locals(1)[0]

	var maxOperations uint = 3
	_, _ = local.Policy().SetMaxOperationsInProposal(maxOperations)

	var seals []seal.Seal
	for i := 0; i < int(maxOperations); i++ {
		var sl seal.Seal
		if i == 1 {
			sl = t.newOperationSeal(local, 2)
		} else {
			sl = t.newOperationSeal(local, 1)
		}
		seals = append(seals, sl)
	}

	t.NoError(local.Storage().NewSeals(seals))

	proposalMaker := NewProposalMaker(local)

	round := base.Round(1)
	proposal, err := proposalMaker.Proposal(round)
	t.NoError(err)

	t.Equal(len(seals)-1, len(proposal.Seals()))

	for _, h := range proposal.Seals() {
		t.False(seals[2].Hash().Equal(h))
	}
}

func (t *testProposalMaker) TestNumberOperationMatch() {
	local := t.locals(1)[0]

	var maxOperations uint = 3
	_, _ = local.Policy().SetMaxOperationsInProposal(maxOperations)

	var seals []seal.Seal
	for i := 0; i < int(maxOperations); i++ {
		sl := t.newOperationSeal(local, 1)
		seals = append(seals, sl)
	}
	t.NoError(local.Storage().NewSeals(seals))

	proposalMaker := NewProposalMaker(local)

	round := base.Round(1)
	proposal, err := proposalMaker.Proposal(round)
	t.NoError(err)

	t.Equal(len(seals), len(proposal.Seals()))
}

func TestProposalMaker(t *testing.T) {
	suite.Run(t, new(testProposalMaker))
}
