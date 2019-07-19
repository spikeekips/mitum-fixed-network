package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/node"
	"github.com/spikeekips/mitum/seal"
)

type testINITBallotBody struct {
	suite.Suite
}

func (t *testINITBallotBody) TestNew() {
	home := node.NewRandomHome()
	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)

	ballot, err := NewINITBallot(
		home.Address(),
		lastBlock.Hash(),
		nextBlock.Height(),
		nextBlock.Hash(),
		Round(1),
		nextBlock.Proposal(),
	)
	t.NoError(err)

	_ = interface{}(ballot).(seal.Seal)

	t.True(home.Address().Equal(ballot.Node()))
	t.True(nextBlock.Hash().Equal(ballot.Block()))
	t.True(lastBlock.Hash().Equal(ballot.LastBlock()))
	t.Equal(Round(1), ballot.Round())
	t.True(nextBlock.Proposal().Equal(ballot.Proposal()))
	t.Equal(BallotType, ballot.Type())

	err = ballot.Sign(home.PrivateKey(), nil)
	t.NoError(err)

	seal, ok := interface{}(ballot).(seal.Seal)
	t.True(ok)

	err = seal.IsValid()
	t.NoError(err)

	t.Equal(BallotType, seal.Type())
}

func (t *testINITBallotBody) TestZeroRound() {
	home := node.NewRandomHome()
	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)

	ballot, err := NewINITBallot(
		home.Address(),
		lastBlock.Hash(),
		nextBlock.Height(),
		nextBlock.Hash(),
		Round(0),
		nextBlock.Proposal(),
	)
	t.NoError(err)

	_ = interface{}(ballot).(seal.Seal)

	err = ballot.Sign(home.PrivateKey(), nil)
	t.NoError(err)

	_ = interface{}(ballot).(seal.Seal)

	err = ballot.IsValid()
	t.Contains(err.Error(), "round should be greater than 0")
}

func TestINITBallotBody(t *testing.T) {
	suite.Run(t, new(testINITBallotBody))
}
