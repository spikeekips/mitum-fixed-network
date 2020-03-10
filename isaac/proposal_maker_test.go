package isaac

import (
	"testing"

	"github.com/spikeekips/mitum/operation"
	"github.com/spikeekips/mitum/valuehash"
	"github.com/stretchr/testify/suite"
)

type testProposalMaker struct {
	baseTestStateHandler
}

func (t *testProposalMaker) TestCached() {
	proposalMaker := NewProposalMaker(t.localstate)

	round := Round(1)
	proposal, err := proposalMaker.Proposal(round)
	t.NoError(err)

	newProposal, err := proposalMaker.Proposal(round)
	t.NoError(err)

	t.True(proposal.Hash().Equal(newProposal.Hash()))
}

func (t *testProposalMaker) TestClean() {
	localstate, _ := t.states()
	proposalMaker := NewProposalMaker(localstate)

	round := Round(1)
	_, err := proposalMaker.Proposal(round)
	t.NoError(err)

	newBlock, err := NewTestBlockV0(localstate.LastBlock().Height()+1, Round(0), nil, valuehash.RandomSHA256())
	t.NoError(err)
	_ = localstate.SetLastBlock(newBlock)

	t.NotNil(proposalMaker.proposed)
}

func (t *testProposalMaker) TestSeals() {
	localstate, _ := t.states()

	var ops []operation.Seal
	// 10 operation.Seal
	for i := 0; i < 10; i++ {
		sl := t.newOperationSeal(localstate)
		t.NoError(localstate.Storage().NewSeal(sl))

		ops = append(ops, sl)
	}

	proposalMaker := NewProposalMaker(localstate)

	round := Round(1)
	proposal, err := proposalMaker.Proposal(round)
	t.NoError(err)

	t.Equal(len(ops), len(proposal.Seals()))

	var expectedSeals []valuehash.Hash
	err = localstate.Storage().StagedOperationSeals(func(sl operation.Seal) (bool, error) {
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
