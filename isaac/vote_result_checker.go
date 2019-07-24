package isaac

import (
	"context"

	"github.com/spikeekips/mitum/common"
	"golang.org/x/xerrors"
)

type VoteResultChecker struct {
	homeState *HomeState
}

func NewJoinVoteResultChecker(homeState *HomeState) *common.ChainChecker {
	vrc := VoteResultChecker{
		homeState: homeState,
	}

	return common.NewChainChecker(
		"vote-result-checker",
		context.Background(),
		vrc.checkFinished,
		vrc.checkHeightAndRound,
		vrc.checkINIT,
		vrc.checkNotINIT,
	)
}

func (vrc VoteResultChecker) checkFinished(c *common.ChainChecker) error {
	var vr VoteResult
	if err := c.ContextValue("vr", &vr); err != nil {
		return err
	}

	if vr.IsClosed() {
		return common.ChainCheckerStopError.Newf("already closed")
	}

	if !vr.IsFinished() {
		return common.ChainCheckerStopError.Newf("not finished")
	}

	return nil
}

func (vrc VoteResultChecker) checkHeightAndRound(c *common.ChainChecker) error {
	var vr VoteResult
	if err := c.ContextValue("vr", &vr); err != nil {
		return err
	}

	lastHeight := vrc.homeState.Block().Height()
	lastRound := vrc.homeState.Block().Round()

	// NOTE VoteResult.Height() should be greater than lastHeight
	if vr.Height().Cmp(lastHeight) < 1 {
		return xerrors.Errorf(
			"VoteResult.Height() should be greater than lastHeight; VoteResult=%q lastHeight=%q",
			vr.Height(),
			lastHeight,
		)
	}

	// NOTE VoteResult.Round() should be greater than lastRound
	if vr.Round() <= lastRound {
		return xerrors.Errorf(
			"VoteResult.Round() should be greater than lastRound; VoteResult=%q lastRound=%q",
			vr.Round(),
			lastRound,
		)
	}

	return nil
}

func (vrc VoteResultChecker) checkINIT(c *common.ChainChecker) error {
	var vr VoteResult
	if err := c.ContextValue("vr", &vr); err != nil {
		return err
	}

	if vr.Stage() != StageINIT {
		return nil
	}

	return nil
}

func (vrc VoteResultChecker) checkNotINIT(c *common.ChainChecker) error {
	var vr VoteResult
	if err := c.ContextValue("vr", &vr); err != nil {
		return err
	}

	if vr.Stage() == StageINIT {
		return nil
	}

	var lastINITVoteResult VoteResult
	if err := c.ContextValue("lastINITVoteResult", &lastINITVoteResult); err != nil {
		return err
	}

	lastHeight := lastINITVoteResult.Height()
	lastRound := lastINITVoteResult.Round()
	if !vr.Height().Equal(lastHeight) {
		return common.ChainCheckerStopError.Newf(
			"VoteResult has different height; last_height=%q height=%q",
			lastHeight,
			vr.Height(),
		)
	}

	if vr.Round() != lastRound {
		return common.ChainCheckerStopError.Newf(
			"VoteResult has different round; last_round=%q round=%q",
			lastRound,
			vr.Round(),
		)
	}

	return nil
}
