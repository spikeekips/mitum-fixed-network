package isaac

import (
	"context"

	"github.com/inconshreveable/log15"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/common"
)

type CompilerBallotChecker struct {
	homeState *HomeState
	suffrage  Suffrage
}

func NewCompilerBallotChecker(homeState *HomeState, suffrage Suffrage) *common.ChainChecker {
	cbc := CompilerBallotChecker{
		homeState: homeState,
		suffrage:  suffrage,
	}

	return common.NewChainChecker(
		"compiler-ballot-checker",
		context.Background(),
		cbc.initialize,
		cbc.checkInSuffrage,
		cbc.checkHeightAndRound,
		cbc.checkINIT,
		cbc.checkNotINIT,
	)
}

func (cbc CompilerBallotChecker) initialize(c *common.ChainChecker) error {
	var ballot Ballot
	if err := c.ContextValue("ballot", &ballot); err != nil {
		return err
	}

	log_ := c.Log().New(log15.Ctx{"ballot": ballot.Hash()})
	_ = c.SetContext("log", log_)

	log_.Debug("will check ballot", "seal", ballot)

	return nil
}

func (cbc CompilerBallotChecker) checkInSuffrage(c *common.ChainChecker) error {
	var ballot Ballot
	if err := c.ContextValue("ballot", &ballot); err != nil {
		return err
	}

	if ballot.Stage() != StageINIT {
		acting := cbc.suffrage.Acting(ballot.Height(), ballot.Round())
		if !acting.Exists(ballot.Node()) {
			return xerrors.Errorf(
				"%s ballot node does not in acting suffrage; ballot=%v node=%v",
				ballot.Stage(),
				ballot.Hash(),
				ballot.Node(),
			)
		}
	} else if !cbc.suffrage.Exists(ballot.Height().Sub(1), ballot.Node()) {
		return xerrors.Errorf(
			"%s ballot node does not in suffrage; ballot=%v node=%v",
			ballot.Stage(),
			ballot.Hash(),
			ballot.Node(),
		)
	}

	return nil
}

func (cbc CompilerBallotChecker) checkHeightAndRound(c *common.ChainChecker) error {
	var ballot Ballot
	if err := c.ContextValue("ballot", &ballot); err != nil {
		return err
	}

	var log_ log15.Logger
	if err := c.ContextValue("log", &log_); err != nil {
		return err
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
			log_.Debug(
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

	var log_ log15.Logger
	if err := c.ContextValue("log", &log_); err != nil {
		return err
	}

	// NOTE ballot.Height() should be greater than homeState.Block().Height()
	if ballot.Height().Cmp(cbc.homeState.Block().Height()) < 1 {
		err := xerrors.Errorf("lower ballot height")
		log_.Error(
			"ballot.Height() should be greater than homeState.Block().Height(); ignore this ballot",
			"ballot_height", ballot.Height(),
			"height", cbc.homeState.Block().Height(),
		)

		return err
	} else {
		sub := ballot.Height().Sub(cbc.homeState.Block().Height())
		switch sub.Int64() {
		case 2:
			if !ballot.LastBlock().Equal(cbc.homeState.Block().Hash()) {
				return xerrors.Errorf(
					"block does not match; ballot=%v block=%v",
					ballot.Block(),
					cbc.homeState.Block().Hash(),
				)
			}
		case 1:
			if !ballot.Block().Equal(cbc.homeState.Block().Hash()) {
				return xerrors.Errorf(
					"block does not match; ballot=%v block=%v",
					ballot.Block(),
					cbc.homeState.Block().Hash(),
				)
			}
			if ballot.LastRound() != cbc.homeState.Block().Round() {
				return xerrors.Errorf(
					"round does not match; ballot=%v round=%v",
					ballot.LastRound(),
					cbc.homeState.Block().Round(),
				)
			}
		default:
			log_.Warn(
				"ballot height is higher than expected; ignore this ballot",
				"ballot_height", ballot.Height(),
				"height", cbc.homeState.Block().Height(),
			)
			return common.ChainCheckerStopError
		}
	}

	var lastINITVoteResult VoteResult
	if err := c.ContextValue("lastINITVoteResult", &lastINITVoteResult); err != nil {
		return err
	}

	if !lastINITVoteResult.IsFinished() {
		log_.Debug("lastINITVoteResult is empty")
		return nil
	}

	lastHeight := lastINITVoteResult.Height()
	lastRound := lastINITVoteResult.Round()

	if ballot.Height().Equal(lastHeight) { // this should be draw; round should be greater
		if ballot.Round() <= lastRound {
			err := xerrors.Errorf("ballot.Round() should be greater than lastINITVoteResult")
			log_.Debug(
				"compared with lastINITVoteResult",
				"last_height", lastHeight,
				"last_round", lastRound,
				"ballot_height", ballot.Height(),
				"ballot_round", ballot.Round(),
				"error", err,
			)
			return err
		}
	} else if ballot.Height().Cmp(lastHeight) < 1 {
		err := xerrors.Errorf("ballot.Height() should be greater than lastINITVoteResult")
		log_.Debug(
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

	var log_ log15.Logger
	if err := c.ContextValue("log", &log_); err != nil {
		return err
	}

	// NOTE ballot.Height() should be greater than homeState.Block().Height()
	if sub := ballot.Height().Sub(cbc.homeState.Block().Height()); sub.Int64() < 1 {
		err := xerrors.Errorf("lower ballot height")
		log_.Error(
			"ballot.Height() should be greater than homeState.Block().Height() + 1; ignore this ballot",
			"ballot_height", ballot.Height(),
			"height", cbc.homeState.Block().Height(),
		)

		return err
	}

	// NOTE ballot.Round() should be greater than homeState.Block().Round()
	if ballot.LastBlock().Equal(cbc.homeState.Block().Hash()) {
		if ballot.LastRound() != cbc.homeState.Block().Round() {
			err := xerrors.Errorf("ballot last round does not match with last block")
			log_.Error(
				"ballot.Round() should be same with homeState.Block().Round(); ignore this ballot",
				"ballot_round", ballot.Round(),
				"round", cbc.homeState.Block().Round(),
			)

			return err
		}
	}

	var lastINITVoteResult VoteResult
	if err := c.ContextValue("lastINITVoteResult", &lastINITVoteResult); err != nil {
		return err
	}

	// NOTE without previous lastINITVoteResult, the stages except init will be
	// ignored
	if !lastINITVoteResult.IsFinished() {
		err := xerrors.Errorf("lastINITVoteResult is empty")
		log_.Error("lastINITVoteResult is empty; ignore this ballot")
		return err
	}

	// NOTE the height of stages except init should be same with
	// lastINITVoteResult.Height()
	if !ballot.Height().Equal(lastINITVoteResult.Height()) {
		err := xerrors.Errorf("lower ballot height")
		log_.Error(
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
		log_.Error(
			"ballot.Round() should be same with last init ballot; ignore this ballot",
			"ballot_round", ballot.Round(),
			"round", lastINITVoteResult.Round(),
		)
		return err
	}

	return nil
}
