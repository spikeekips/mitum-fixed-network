package isaac

import (
	"testing"

	"github.com/spikeekips/mitum/valuehash"
	"github.com/stretchr/testify/suite"
)

type testProposalMaker struct {
	baseTestStateHandler
}

func (t *testProposalMaker) TestCached() {
	proposalMaker := NewProposalMaker(t.localstate)

	round := Round(1)
	proposal, err := proposalMaker.Proposal(round, nil)
	t.NoError(err)

	newProposal, err := proposalMaker.Proposal(round, nil)
	t.NoError(err)

	t.True(proposal.Hash().Equal(newProposal.Hash()))
}

func (t *testProposalMaker) TestClean() {
	localstate, _ := t.states()
	proposalMaker := NewProposalMaker(localstate)

	round := Round(1)
	_, err := proposalMaker.Proposal(round, nil)
	t.NoError(err)

	newBlock, err := NewTestBlockV0(localstate.LastBlock().Height()+1, Round(0), nil, valuehash.RandomSHA256())
	t.NoError(err)
	_ = localstate.SetLastBlock(newBlock)

	t.Equal(1, len(proposalMaker.proposed))

	proposalMaker.Clean()

	t.Equal(0, len(proposalMaker.proposed))
}

func TestProposalMaker(t *testing.T) {
	suite.Run(t, new(testProposalMaker))
}
