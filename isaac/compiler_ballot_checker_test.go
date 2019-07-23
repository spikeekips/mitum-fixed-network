package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/common"
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
		nextBlock.Height(),
		nextBlock.Hash(),
		nextBlock.Round(),
		lastBlock.Proposal(),
	)

	checker := NewCompilerBallotChecker(homeState)
	err := checker.
		New(nil).
		SetContext(
			"ballot", ballot,
			"lastINITVoteResult", VoteResult{},
			"lastStagesVoteResult", VoteResult{},
		).Check()
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
		lastBlock.Height(),
		nextBlock.Hash(),
		nextBlock.Round(),
		lastBlock.Proposal(),
	)

	checker := NewCompilerBallotChecker(homeState)
	err := checker.
		New(nil).
		SetContext(
			"ballot", ballot,
			"lastINITVoteResult", VoteResult{},
			"lastStagesVoteResult", VoteResult{},
		).Check()
	t.Contains(err.Error(), "lower ballot height")
}

func (t *testCompilerBallotChecker) TestINITBallotRoundNotHigherThanHomeState() {
	home := node.NewRandomHome()
	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)

	homeState := NewHomeState(home, lastBlock)

	ballot, _ := NewINITBallot(
		home.Address(),
		lastBlock.Hash(),
		nextBlock.Height(),
		nextBlock.Hash(),
		lastBlock.Round(),
		lastBlock.Proposal(),
	)

	checker := NewCompilerBallotChecker(homeState)
	err := checker.
		New(nil).
		SetContext(
			"ballot", ballot,
			"lastINITVoteResult", VoteResult{},
			"lastStagesVoteResult", VoteResult{},
		).Check()
	t.Contains(err.Error(), "lower ballot round")
}

func (t *testCompilerBallotChecker) TestINITBallotHeightNotHigherThanLastINITVoteResult() {
	home := node.NewRandomHome()
	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)

	homeState := NewHomeState(home, lastBlock)

	ballot, _ := NewINITBallot(
		home.Address(),
		lastBlock.Hash(),
		nextBlock.Height(),
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
		New(nil).
		SetContext(
			"ballot", ballot,
			"lastINITVoteResult", lastINITVoteResult,
			"lastStagesVoteResult", VoteResult{},
		).Check()
	t.Contains(err.Error(), "lower ballot height")
}

func (t *testCompilerBallotChecker) TestSIGNBallotHeightNotSameWithLastINITVoteResult() {
	defer common.DebugPanic()

	home := node.NewRandomHome()
	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)

	homeState := NewHomeState(home, lastBlock)

	ballot, _ := NewSIGNBallot(
		home.Address(),
		lastBlock.Hash(),
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
		New(nil).
		SetContext(
			"ballot", ballot,
			"lastINITVoteResult", lastINITVoteResult,
			"lastStagesVoteResult", VoteResult{},
		).Check()
	t.Contains(err.Error(), "lower ballot height")
}

func (t *testCompilerBallotChecker) TestSIGNBallotRoundNotSameWithLastINITVoteResult() {
	defer common.DebugPanic()

	home := node.NewRandomHome()
	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)

	homeState := NewHomeState(home, lastBlock)

	ballot, _ := NewSIGNBallot(
		home.Address(),
		lastBlock.Hash(),
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
		New(nil).
		SetContext(
			"ballot", ballot,
			"lastINITVoteResult", lastINITVoteResult,
			"lastStagesVoteResult", VoteResult{},
		).Check()
	t.Contains(err.Error(), "lower ballot round")
}

func TestCompilerBallotChecker(t *testing.T) {
	suite.Run(t, new(testCompilerBallotChecker))
}
