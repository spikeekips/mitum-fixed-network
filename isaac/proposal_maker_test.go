package isaac

import (
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testProposalMaker struct {
	BaseTest
}

func (t *testProposalMaker) TestCached() {
	local := t.Locals(1)[0]

	proposalMaker := NewProposalMaker(local.Node(), local.Database(), local.Policy())

	ib := t.NewINITBallot(local, base.Round(0), nil)
	initFact := ib.INITFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, local)
	t.NoError(err)

	proposal, err := proposalMaker.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	newProposal, err := proposalMaker.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	t.True(proposal.Hash().Equal(newProposal.Hash()))
}

func (t *testProposalMaker) TestClean() {
	local := t.Locals(1)[0]

	proposalMaker := NewProposalMaker(local.Node(), local.Database(), local.Policy())

	height := base.Height(33)
	round := base.Round(1)
	_, err := proposalMaker.Proposal(height, round, nil)
	t.NoError(err)

	t.NotNil(proposalMaker.proposed)
}

func (t *testProposalMaker) TestSeals() {
	local := t.Locals(1)[0]

	var seals []seal.Seal
	// 10 operation.Seal
	for i := 0; i < 10; i++ {
		sl, _ := t.NewOperationSeal(local, 1)

		seals = append(seals, sl)
	}
	t.NoError(local.Database().NewSeals(seals))

	proposalMaker := NewProposalMaker(local.Node(), local.Database(), local.Policy())

	height := base.Height(33)
	round := base.Round(1)
	proposal, err := proposalMaker.Proposal(height, round, nil)
	t.NoError(err)

	t.Equal(len(seals), len(proposal.Seals()))

	var expectedSeals []valuehash.Hash
	err = local.Database().StagedOperationSeals(func(sl operation.Seal) (bool, error) {
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
	local := t.Locals(1)[0]

	var maxOperations uint = 3
	_, _ = local.Policy().SetMaxOperationsInProposal(maxOperations)

	var seals []seal.Seal
	for i := 0; i < int(maxOperations-1); i++ {
		sl, _ := t.NewOperationSeal(local, 1)
		seals = append(seals, sl)
	}

	over, _ := t.NewOperationSeal(local, 2)
	seals = append(seals, over)

	t.NoError(local.Database().NewSeals(seals))

	proposalMaker := NewProposalMaker(local.Node(), local.Database(), local.Policy())

	height := base.Height(33)
	round := base.Round(1)
	proposal, err := proposalMaker.Proposal(height, round, nil)
	t.NoError(err)

	t.Equal(len(seals)-1, len(proposal.Seals()))

	for _, h := range proposal.Seals() {
		t.False(over.Hash().Equal(h))
	}
}

func (t *testProposalMaker) TestOneSealOver1() {
	local := t.Locals(1)[0]

	var maxOperations uint = 3
	_, _ = local.Policy().SetMaxOperationsInProposal(maxOperations)

	var seals []seal.Seal
	for i := 0; i < int(maxOperations); i++ {
		var sl seal.Seal
		if i == 1 {
			sl, _ = t.NewOperationSeal(local, 2)
		} else {
			sl, _ = t.NewOperationSeal(local, 1)
		}
		seals = append(seals, sl)
	}

	t.NoError(local.Database().NewSeals(seals))

	proposalMaker := NewProposalMaker(local.Node(), local.Database(), local.Policy())

	height := base.Height(33)
	round := base.Round(1)
	proposal, err := proposalMaker.Proposal(height, round, nil)
	t.NoError(err)

	t.Equal(len(seals)-1, len(proposal.Seals()))

	for _, h := range proposal.Seals() {
		t.False(seals[2].Hash().Equal(h))
	}
}

func (t *testProposalMaker) TestNumberOperationMatch() {
	local := t.Locals(1)[0]

	var maxOperations uint = 3
	_, _ = local.Policy().SetMaxOperationsInProposal(maxOperations)

	var seals []seal.Seal
	for i := 0; i < int(maxOperations); i++ {
		sl, _ := t.NewOperationSeal(local, 1)
		seals = append(seals, sl)
	}
	t.NoError(local.Database().NewSeals(seals))

	proposalMaker := NewProposalMaker(local.Node(), local.Database(), local.Policy())

	height := base.Height(33)
	round := base.Round(1)
	proposal, err := proposalMaker.Proposal(height, round, nil)
	t.NoError(err)

	t.Equal(len(seals), len(proposal.Seals()))
}

func TestProposalMaker(t *testing.T) {
	suite.Run(t, new(testProposalMaker))
}
