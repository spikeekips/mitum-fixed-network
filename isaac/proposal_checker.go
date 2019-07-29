package isaac

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/common"
)

type ProposalChecker struct {
	homeState *HomeState
}

func NewProposalCheckerBooting(homeState *HomeState) *common.ChainChecker {
	pc := ProposalChecker{
		homeState: homeState,
	}

	return common.NewChainChecker(
		"booting-proposal-checker",
		context.Background(),
		pc.saveProposal,
		pc.checkHeightAndRoundWithHomeState,
		pc.checkProposerIsValid,
	)
}

func NewProposalCheckerJoin(homeState *HomeState) *common.ChainChecker {
	pc := ProposalChecker{
		homeState: homeState,
	}

	return common.NewChainChecker(
		"join-proposal-checker",
		context.Background(),
		pc.saveProposal,
		pc.checkHeightAndRoundWithHomeState,
		pc.checkHeightAndRoundWithLastINITVoteResult,
		pc.checkProposerIsValid,
	)
}

func NewProposalCheckerConsensus(homeState *HomeState) *common.ChainChecker {
	pc := ProposalChecker{
		homeState: homeState,
	}

	return common.NewChainChecker(
		"join-proposal-checker",
		context.Background(),
		pc.saveProposal,
		pc.checkHeightAndRoundWithHomeState,
		pc.checkHeightAndRoundWithLastINITVoteResult,
		pc.checkProposerIsValid,
	)
}

func (pc ProposalChecker) saveProposal(*common.ChainChecker) error {
	// TODO

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
		c.Log().Error(
			"proposal.Height() should be same than homeState.Block().Height() + 1; ignore this ballot",
			"proposal_height", proposal.Height(),
			"expected_height", pc.homeState.Block().Height().Add(1),
			"current_height", pc.homeState.Block().Height(),
		)

		return err
	}

	// NOTE proposal.Round() should be greater than homeState.Block().Round()
	if proposal.Round() <= pc.homeState.Block().Round() {
		err := xerrors.Errorf("invalid proposal round")
		c.Log().Error(
			"proposal.Round() should be greater than homeState.Block().Round(); ignore this ballot",
			"proposal_round", proposal.Round(),
			"expected_round", pc.homeState.Block().Round()+1,
		)

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
		c.Log().Error(
			"proposal.Height() should be same with lastINITVoteResult; ignore this ballot",
			"proposal_height", proposal.Height(),
			"height", lastINITVoteResult.Height(),
		)

		return err
	}

	// NOTE proposal.Round() should be same with lastINITVoteResult
	if proposal.Round() != lastINITVoteResult.Round() {
		err := xerrors.Errorf("invalid proposal round")
		c.Log().Error(
			"proposal.Round() should be same with lastINITVoteResult; ignore this ballot",
			"proposal_round", proposal.Round(),
			"round", lastINITVoteResult.Round(),
		)

		return err
	}

	return nil
}

func (pc ProposalChecker) checkProposerIsValid(*common.ChainChecker) error {
	// TODO

	return nil
}
