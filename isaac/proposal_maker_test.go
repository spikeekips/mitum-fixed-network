package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
)

type testProposalMaker struct {
	baseTestStateHandler
}

func (t *testProposalMaker) TestCached() {
	proposalMaker := NewProposalMaker(t.localstates(1)[0])

	round := base.Round(1)
	proposal, err := proposalMaker.Proposal(round)
	t.NoError(err)

	newProposal, err := proposalMaker.Proposal(round)
	t.NoError(err)

	t.True(proposal.Hash().Equal(newProposal.Hash()))
}

func (t *testProposalMaker) TestClean() {
	local := t.localstates(1)[0]

	proposalMaker := NewProposalMaker(local)

	round := base.Round(1)
	_, err := proposalMaker.Proposal(round)
	t.NoError(err)

	t.NotNil(proposalMaker.proposed)
}

func (t *testProposalMaker) TestSeals() {
	local := t.localstates(1)[0]

	var ops []operation.Seal
	var seals []seal.Seal
	// 10 operation.Seal
	for i := 0; i < 10; i++ {
		sl := t.newOperationSeal(local)

		ops = append(ops, sl)
		seals = append(seals, sl)
	}
	t.NoError(local.Storage().NewSeals(seals))

	proposalMaker := NewProposalMaker(local)

	round := base.Round(1)
	proposal, err := proposalMaker.Proposal(round)
	t.NoError(err)

	t.Equal(len(ops), len(proposal.Seals()))

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

func TestProposalMaker(t *testing.T) {
	suite.Run(t, new(testProposalMaker))
}
