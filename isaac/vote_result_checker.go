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
