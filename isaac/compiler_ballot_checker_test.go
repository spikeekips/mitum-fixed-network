package isaac

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/node"
)

type testCompilerBallotChecker struct {
	suite.Suite
}

func (t *testCompilerBallotChecker) TestEmptyLastVoteResult() {
	home := node.NewRandomHome()
	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)

	homeState := NewHomeState(home, lastBlock)

	ballot, _ := NewINITBallot(
		home.Address(),
		lastBlock.Hash(),
		lastBlock.Round(),
		nextBlock.Height(),
		nextBlock.Hash(),
		nextBlock.Round(),
		lastBlock.Proposal(),
	)

	checker := NewCompilerBallotChecker(homeState)
	err := checker.
		New(context.TODO()).
		SetContext("ballot", ballot).
		SetContext("lastINITVoteResult", VoteResult{}).
		SetContext("lastStagesVoteResult", VoteResult{}).
		Check()
	t.NoError(err)
}

func (t *testCompilerBallotChecker) TestINITBallotHeightNotHigherThanHomeState() {
	home := node.NewRandomHome()
	prevBlock := NewRandomBlock()
	lastBlock := NewRandomNextBlock(prevBlock)
	nextBlock := NewRandomNextBlock(lastBlock)

	homeState := NewHomeState(home, lastBlock)

	ballot, _ := NewINITBallot(
		home.Address(),
		lastBlock.Hash(),
		lastBlock.Round(),
		prevBlock.Height(),
		nextBlock.Hash(),
		nextBlock.Round(),
		lastBlock.Proposal(),
	)

	checker := NewCompilerBallotChecker(homeState)
	err := checker.
		New(context.TODO()).
		SetContext("ballot", ballot).
		SetContext("lastINITVoteResult", VoteResult{}).
		SetContext("lastStagesVoteResult", VoteResult{}).
		Check()
	t.Contains(err.Error(), "lower ballot height")
}

func (t *testCompilerBallotChecker) TestINITBallotHeightNotHigherThanLastINITVoteResult() {
	home := node.NewRandomHome()
	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)

	homeState := NewHomeState(home, lastBlock)

	ballot, _ := NewINITBallot(
		home.Address(),
		lastBlock.Hash(),
		lastBlock.Round(),
		lastBlock.Height(),
		nextBlock.Hash(),
		nextBlock.Round(),
		lastBlock.Proposal(),
	)

	lastINITVoteResult := NewVoteResult(
		nextBlock.Height().Add(1),
		nextBlock.Round(),
		StageINIT,
	)
	lastINITVoteResult = lastINITVoteResult.
		SetAgreement(Majority)

	checker := NewCompilerBallotChecker(homeState)
	err := checker.
		New(context.TODO()).
		SetContext("ballot", ballot).
		SetContext("lastINITVoteResult", lastINITVoteResult).
		SetContext("lastStagesVoteResult", VoteResult{}).
		Check()
	t.Contains(err.Error(), "lower ballot height")
}

func (t *testCompilerBallotChecker) TestSIGNBallotHeightNotSameWithLastINITVoteResult() {
	home := node.NewRandomHome()
	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)

	homeState := NewHomeState(home, lastBlock)

	ballot, _ := NewSIGNBallot(
		home.Address(),
		lastBlock.Hash(),
		lastBlock.Round(),
		nextBlock.Height(),
		nextBlock.Hash(),
		nextBlock.Round(),
		nextBlock.Proposal(),
	)

	lastINITVoteResult := NewVoteResult(
		lastBlock.Height(),
		nextBlock.Round(),
		StageINIT,
	)
	lastINITVoteResult = lastINITVoteResult.
		SetAgreement(Majority)

	checker := NewCompilerBallotChecker(homeState)
	err := checker.
		New(context.TODO()).
		SetContext("ballot", ballot).
		SetContext("lastINITVoteResult", lastINITVoteResult).
		SetContext("lastStagesVoteResult", VoteResult{}).
		Check()
	t.Contains(err.Error(), "lower ballot height")
}

func (t *testCompilerBallotChecker) TestSIGNBallotRoundNotSameWithLastINITVoteResult() {
	home := node.NewRandomHome()
	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)

	homeState := NewHomeState(home, lastBlock)

	ballot, _ := NewSIGNBallot(
		home.Address(),
		lastBlock.Hash(),
		lastBlock.Round(),
		nextBlock.Height(),
		nextBlock.Hash(),
		nextBlock.Round(),
		nextBlock.Proposal(),
	)

	lastINITVoteResult := NewVoteResult(
		nextBlock.Height(),
		nextBlock.Round()-1,
		StageINIT,
	)
	lastINITVoteResult = lastINITVoteResult.
		SetAgreement(Majority)

	checker := NewCompilerBallotChecker(homeState)
	err := checker.
		New(context.TODO()).
		SetContext("ballot", ballot).
		SetContext("lastINITVoteResult", lastINITVoteResult).
		SetContext("lastStagesVoteResult", VoteResult{}).
		Check()
	t.Contains(err.Error(), "lower ballot round")
}

func TestCompilerBallotChecker(t *testing.T) {
	suite.Run(t, new(testCompilerBallotChecker))
}
