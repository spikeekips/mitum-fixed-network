package isaac

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/common"
)

type CompilerBallotChecker struct {
	homeState *HomeState
}

func NewCompilerBallotChecker(homeState *HomeState) *common.ChainChecker {
	cbc := CompilerBallotChecker{
		homeState: homeState,
	}

	return common.NewChainChecker(
		"compiler-ballot-checker",
		context.Background(),
		cbc.checkHeightAndRound,
		cbc.checkINIT,
		cbc.checkNotINIT,
	)
}

func (cbc CompilerBallotChecker) checkHeightAndRound(c *common.ChainChecker) error {
	var ballot Ballot
	if err := c.ContextValue("ballot", &ballot); err != nil {
		return err
	}

	// NOTE ballot.Height() should be greater than homeState.Block().Height()
	if ballot.Height().Cmp(cbc.homeState.Block().Height()) < 1 {
		err := xerrors.Errorf("lower ballot height")
		c.Log().Error(
			"ballot.Height() should be greater than homeState.Block().Height(); ignore this ballot",
			"ballot_height", ballot.Height(),
			"height", cbc.homeState.Block().Height(),
		)

		return err
	}

	// NOTE ballot.Round() should be greater than homeState.Block().Round()
	if ballot.LastBlock().Equal(cbc.homeState.Block().Hash()) {
		if ballot.LastRound() != cbc.homeState.Block().Round() {
			err := xerrors.Errorf("ballot last round does not match with last block")
			c.Log().Error(
				"ballot.Round() should be same with homeState.Block().Round(); ignore this ballot",
				"ballot_round", ballot.Round(),
				"round", cbc.homeState.Block().Round(),
			)

			return err
		}
	}

	// NOTE ballot.Height() and ballot.Round() should be same than last init ballot
	var lastINITVoteResult VoteResult
	if err := c.ContextValue("lastINITVoteResult", &lastINITVoteResult); err != nil {
		return err
	}

	// NOTE lastINITVoteResult is not empty, ballot.Height() should be same or
	// greater than lastINITVoteResult.Height()
	if lastINITVoteResult.IsFinished() {
		if ballot.Height().Cmp(lastINITVoteResult.Height()) < 0 {
			err := xerrors.Errorf("lower ballot height")
			c.Log().Error(
				"ballot.Height() should be greater than last init ballot; ignore this ballot",
				"ballot_height", ballot.Height(),
				"height", lastINITVoteResult.Height(),
			)
			return err
		}
	}

	return nil
}

func (cbc CompilerBallotChecker) checkINIT(c *common.ChainChecker) error {
	var ballot Ballot
	if err := c.ContextValue("ballot", &ballot); err != nil {
		return err
	}

	if ballot.Stage() != StageINIT {
		return nil
	}

	var lastINITVoteResult VoteResult
	if err := c.ContextValue("lastINITVoteResult", &lastINITVoteResult); err != nil {
		return err
	}

	if !lastINITVoteResult.IsFinished() {
		c.Log().Debug("lastINITVoteResult is empty; ignore this ballot")
		return nil
	}

	lastHeight := lastINITVoteResult.Height()
	lastRound := lastINITVoteResult.Round()

	if ballot.Height().Equal(lastHeight) { // this should be draw; round should be greater
		if ballot.Round() <= lastRound {
			err := xerrors.Errorf("ballot.Round() should be greater than lastINITVoteResult")
			c.Log().Error(
				"compared with lastINITVoteResult",
				"last_height", lastHeight,
				"last_round", lastRound,
				"ballot_height", ballot.Height(),
				"ballot_round", ballot.Round(),
				"error", err,
			)
			return err
		}
	} else if ballot.Height().Cmp(lastHeight) <= 1 {
		err := xerrors.Errorf("ballot.Height() should be greater than lastINITVoteResult")
		c.Log().Error(
			"compared with lastINITVoteResult",
			"last_height", lastHeight,
			"last_round", lastRound,
			"ballot_height", ballot.Height(),
			"ballot_round", ballot.Round(),
			"error", err,
		)
		return err
	}

	return nil
}

func (cbc CompilerBallotChecker) checkNotINIT(c *common.ChainChecker) error {
	var ballot Ballot
	if err := c.ContextValue("ballot", &ballot); err != nil {
		return err
	}

	switch ballot.Stage() {
	case StageINIT:
		return nil
	}

	var lastINITVoteResult VoteResult
	if err := c.ContextValue("lastINITVoteResult", &lastINITVoteResult); err != nil {
		return err
	}

	// NOTE without previous lastINITVoteResult, the stages except init will be
	// ignored
	if !lastINITVoteResult.IsFinished() {
		err := xerrors.Errorf("lastINITVoteResult is empty")
		c.Log().Error("lastINITVoteResult is empty; ignore this ballot")
		return err
	}

	// NOTE the height of stages except init should be same with
	// lastINITVoteResult.Height()
	if !ballot.Height().Equal(lastINITVoteResult.Height()) {
		err := xerrors.Errorf("lower ballot height")
		c.Log().Error(
			"ballot.Height() should be same with last init ballot; ignore this ballot",
			"ballot_height", ballot.Height(),
			"height", lastINITVoteResult.Height(),
		)
		return err
	}

	// NOTE the round of stages except init should be same with
	// lastINITVoteResult.Round()
	if ballot.Round() != lastINITVoteResult.Round() {
		err := xerrors.Errorf("lower ballot round")
		c.Log().Error(
			"ballot.Round() should be same with last init ballot; ignore this ballot",
			"ballot_round", ballot.Round(),
			"round", lastINITVoteResult.Round(),
		)
		return err
	}

	return nil
}