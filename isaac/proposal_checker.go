package isaac

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/common"
)

type ProposalChecker struct {
	homeState *HomeState
	suffrage  Suffrage
}

func NewProposalCheckerBooting(homeState *HomeState) *common.ChainChecker {
	pc := ProposalChecker{
		homeState: homeState,
	}

	return common.NewChainChecker(
		"booting-proposal-checker",
		context.Background(),
		pc.checkInActing,
		pc.checkHeightAndRoundWithHomeState,
	)
}

func NewProposalCheckerJoin(homeState *HomeState, suffrage Suffrage) *common.ChainChecker {
	pc := ProposalChecker{
		homeState: homeState,
		suffrage:  suffrage,
	}

	return common.NewChainChecker(
		"join-proposal-checker",
		context.Background(),
		pc.checkInActing,
		pc.checkHeightAndRoundWithHomeState,
		pc.checkHeightAndRoundWithLastINITVoteResult,
	)
}

func NewProposalCheckerConsensus(homeState *HomeState, suffrage Suffrage) *common.ChainChecker {
	pc := ProposalChecker{
		homeState: homeState,
		suffrage:  suffrage,
	}

	return common.NewChainChecker(
		"join-proposal-checker",
		context.Background(),
		pc.checkInActing,
		pc.checkHeightAndRoundWithHomeState,
		pc.checkHeightAndRoundWithLastINITVoteResult,
	)
}

func (pc ProposalChecker) checkInActing(c *common.ChainChecker) error {
	var proposal Proposal
	if err := c.ContextValue("proposal", &proposal); err != nil {
		return err
	}

	acting := pc.suffrage.Acting(proposal.Height(), proposal.Round())
	if !acting.Proposer().Address().Equal(proposal.Proposer()) {
		return xerrors.Errorf(
			"invalid proposer in proposal; expected=%v proposal=%v",
			acting.Proposer().Address(),
			proposal.Proposer(),
		)
	}

	return nil
}

func (pc ProposalChecker) checkHeightAndRoundWithHomeState(c *common.ChainChecker) error {
	var proposal Proposal
	if err := c.ContextValue("proposal", &proposal); err != nil {
		return err
	}

	// NOTE proposal.Height() should be equal with homeState.Block().Height() +
	// 1
	if !proposal.Height().Equal(pc.homeState.Block().Height().Add(1)) {
		err := xerrors.Errorf("invalid proposal height")
		c.Log().Error().
			Uint64("proposal_height", proposal.Height().Uint64()).
			Uint64("expected_height", pc.homeState.Block().Height().Add(1).Uint64()).
			Uint64("current_height", pc.homeState.Block().Height().Uint64()).
			Msg("proposal.Height() should be same than homeState.Block().Height() + 1; ignore this ballot")

		return err
	}

	return nil
}

func (pc ProposalChecker) checkHeightAndRoundWithLastINITVoteResult(c *common.ChainChecker) error {
	var proposal Proposal
	if err := c.ContextValue("proposal", &proposal); err != nil {
		return err
	}

	var lastINITVoteResult VoteResult
	if err := c.ContextValue("lastINITVoteResult", &lastINITVoteResult); err != nil {
		return err
	}

	if !lastINITVoteResult.IsFinished() {
		return xerrors.Errorf("lastINITVoteResult is empty")
	}

	// NOTE proposal.Height() should be same with lastINITVoteResult
	if !proposal.Height().Equal(lastINITVoteResult.Height()) {
		err := xerrors.Errorf("invalid proposal height")
		c.Log().Error().
			Uint64("proposal_height", proposal.Height().Uint64()).
			Uint64("height", lastINITVoteResult.Height().Uint64()).
			Msg("proposal.Height() should be same with lastINITVoteResult; ignore this ballot")

		return err
	}

	// NOTE proposal.Round() should be same with lastINITVoteResult
	if proposal.Round() != lastINITVoteResult.Round() {
		err := xerrors.Errorf("invalid proposal round")
		c.Log().Error().
			Uint64("proposal_round", proposal.Round().Uint64()).
			Uint64("round", lastINITVoteResult.Round().Uint64()).
			Msg("proposal.Round() should be same with lastINITVoteResult; ignore this ballot")

		return err
	}

	return nil
}
