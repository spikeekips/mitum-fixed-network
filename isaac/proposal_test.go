package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/node"
	"github.com/spikeekips/mitum/seal"
)

type testProposal struct {
	suite.Suite
}

func (t *testProposal) TestNew() {
	home := node.NewRandomHome()
	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)

	proposal, err := NewProposal(
		nextBlock.Height(),
		nextBlock.Round(),
		lastBlock.Hash(),
		home.Address(),
		nil,
	)
	t.NoError(err)

	_ = interface{}(proposal).(seal.Seal)

	t.Equal(ProposalType, proposal.Type())

	t.True(nextBlock.Height().Equal(proposal.Height()))
	t.Equal(nextBlock.Round(), proposal.Round())
	t.True(lastBlock.Hash().Equal(proposal.LastBlock()))
	t.True(home.Address().Equal(proposal.Proposer()))

	err = proposal.Sign(home.PrivateKey(), nil)
	t.NoError(err)

	seal, ok := interface{}(proposal).(seal.Seal)
	t.True(ok)

	err = seal.IsValid()
	t.NoError(err)

	t.Equal(ProposalType, seal.Type())
}

func TestProposal(t *testing.T) {
	suite.Run(t, new(testProposal))
}
