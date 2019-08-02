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
		nextBlock.Hash(),
		nextBlock.Round(),
		nextBlock.Proposal(),
		nextBlock.Height().Add(1),
		Round(1),
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
	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)

	homeState := NewHomeState(home, lastBlock)

	ballot, _ := NewINITBallot(
		home.Address(),
		lastBlock.Hash(),
		nextBlock.Hash(),
		nextBlock.Round(),
		nextBlock.Proposal(),
		nextBlock.Height().Sub(1),
		Round(0),
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

func (t *testCompilerBallotChecker) TestINITBallotHeightLowerThanLastINITVoteResult() {
	home := node.NewRandomHome()
	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)

	homeState := NewHomeState(home, lastBlock)

	lastINITVoteResult := NewVoteResult(
		lastBlock.Height(),
		lastBlock.Round(),
		StageINIT,
	)
	lastINITVoteResult = lastINITVoteResult.
		SetAgreement(Majority)

	ballot, _ := NewINITBallot(
		home.Address(),
		lastBlock.Hash(),
		nextBlock.Hash(),
		nextBlock.Round(),
		nextBlock.Proposal(),
		nextBlock.Height().Sub(1),
		Round(0),
	)

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
