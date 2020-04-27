package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
)

type testProposalMaker struct {
	baseTestStateHandler
}

func (t *testProposalMaker) TestCached() {
	proposalMaker := NewProposalMaker(t.localstate)

	round := base.Round(1)
	proposal, err := proposalMaker.Proposal(round)
	t.NoError(err)

	newProposal, err := proposalMaker.Proposal(round)
	t.NoError(err)

	t.True(proposal.Hash().Equal(newProposal.Hash()))
}

func (t *testProposalMaker) TestClean() {
	localstate, rn0 := t.states()
	defer t.closeStates(localstate, rn0)

	proposalMaker := NewProposalMaker(localstate)

	round := base.Round(1)
	_, err := proposalMaker.Proposal(round)
	t.NoError(err)

	newBlock, err := block.NewTestBlockV0(localstate.LastBlock().Height()+1, base.Round(0), nil, valuehash.RandomSHA256())
	t.NoError(err)
	_ = localstate.SetLastBlock(newBlock)

	t.NotNil(proposalMaker.proposed)
}

func (t *testProposalMaker) TestSeals() {
	localstate, rn0 := t.states()
	defer t.closeStates(localstate, rn0)

	var ops []operation.Seal
	var seals []seal.Seal
	// 10 operation.Seal
	for i := 0; i < 10; i++ {
		sl := t.newOperationSeal(localstate)

		ops = append(ops, sl)
		seals = append(seals, sl)
	}
	t.NoError(localstate.Storage().NewSeals(seals))

	proposalMaker := NewProposalMaker(localstate)

	round := base.Round(1)
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
