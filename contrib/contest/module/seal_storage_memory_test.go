package contest_module

import (
	"testing"

	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/node"
	"github.com/stretchr/testify/suite"
)

type testMemorySealStorage struct {
	suite.Suite
}

func (t *testMemorySealStorage) TestGetProposal() {
	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)
	homeState := isaac.NewHomeState(node.NewRandomHome(), lastBlock).SetBlock(nextBlock)

	newBlock := NewRandomNextBlock(nextBlock)

	pm := isaac.NewDefaultProposalMaker(homeState.Home(), 0)
	proposal, err := pm.Make(
		newBlock.Height(),
		newBlock.Round(),
		nextBlock.Hash(),
	)
	t.NoError(err)

	ss := NewMemorySealStorage()
	t.False(ss.Has(proposal.Hash()))
	{
		sl, found := ss.Get(proposal.Hash())
		t.False(found)
		t.Empty(sl)
	}
	{
		sl, found := ss.GetProposal(proposal.Proposer(), proposal.Height(), proposal.Round())
		t.False(found)
		t.Empty(sl)
	}

	t.NoError(ss.Save(proposal))
	t.True(ss.Has(proposal.Hash()))
	{
		sl, found := ss.Get(proposal.Hash())
		t.True(found)
		t.True(proposal.Hash().Equal(sl.Hash()))
	}
	{
		sl, found := ss.GetProposal(proposal.Proposer(), proposal.Height(), proposal.Round())
		t.True(found)
		t.True(proposal.Hash().Equal(sl.Hash()))
	}
}

func TestMemorySealStorage(t *testing.T) {
	suite.Run(t, new(testMemorySealStorage))
}
