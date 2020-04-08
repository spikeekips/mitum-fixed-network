package isaac

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/logging"
)

type ProposalValidationChecker struct {
	*logging.Logging
	localstate *Localstate
	suffrage   Suffrage
	proposal   Proposal
}

func NewProposalValidationChecker(
	localstate *Localstate, suffrage Suffrage, proposal Proposal,
) *ProposalValidationChecker {
	return &ProposalValidationChecker{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.
				Str("module", "proposal-validation-checker").
				Dict("proposal", logging.Dict().
					Hinted("hash", proposal.Hash()).
					Hinted("height", proposal.Height()).
					Hinted("round", proposal.Round()).
					Hinted("node", proposal.Node()),
				)
		}),
		localstate: localstate,
		suffrage:   suffrage,
		proposal:   proposal,
	}
}

// IsKnown checks proposal is already received; if found, no nore checks.
func (pvc *ProposalValidationChecker) IsKnown() (bool, error) {
	height := pvc.proposal.Height()
	round := pvc.proposal.Round()

	if _, err := pvc.localstate.Storage().Proposal(height, round); err != nil {
		return false, err
	} else {
		return false, nil
	}
}

func (pvc *ProposalValidationChecker) IsProposer() (bool, error) {
	height := pvc.proposal.Height()
	round := pvc.proposal.Round()
	node := pvc.proposal.Node()

	if pvc.suffrage.IsProposer(height, round, node) {
		return true, nil
	}

	err := xerrors.Errorf("proposal has wrong proposer")

	pvc.Log().Error().Err(err).
		Hinted("expected_proposer", pvc.suffrage.Acting(height, round).Proposer().Address()).
		Send()

	pvc.Log().Error().Err(err).Msg("wrong proposer found")

	return false, err
}

func (pvc *ProposalValidationChecker) SaveProposal() (bool, error) {
	if err := pvc.localstate.Storage().NewProposal(pvc.proposal); err != nil {
		return false, err
	}

	return true, nil
}

func (pvc *ProposalValidationChecker) IsOld() (bool, error) {
	height := pvc.proposal.Height()
	round := pvc.proposal.Round()

	ivp := pvc.localstate.LastINITVoteproof()
	if height != ivp.Height() || round != ivp.Round() {
		err := xerrors.Errorf("old Proposal received")
		pvc.Log().Error().Err(err).
			Dict("current", logging.Dict().
				Hinted("height", ivp.Height()).
				Hinted("round", ivp.Round()),
			).
			Msg("old proposal received")

		return false, err
	}

	return true, nil
}
