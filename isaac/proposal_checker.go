package isaac

import (
	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/logging"
)

type ProposalValidationChecker struct {
	*logging.Logger
	localstate *Localstate
	suffrage   Suffrage
	proposal   Proposal
}

func NewProposalValidationChecker(
	localstate *Localstate, suffrage Suffrage, proposal Proposal,
) *ProposalValidationChecker {
	return &ProposalValidationChecker{
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.
				Str("module", "proposal-validation-checker").
				Dict("proposal", zerolog.Dict().
					Str("hash", proposal.Hash().String()).
					Int64("height", proposal.Height().Int64()).
					Uint64("round", proposal.Round().Uint64()).
					Str("node", proposal.Node().String()),
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
		Str("expected_proposer", pvc.suffrage.Acting(height, round).Proposer().Address().String()).
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
			Dict("current", zerolog.Dict().
				Int64("height", ivp.Height().Int64()).
				Uint64("round", ivp.Round().Uint64()),
			).
			Msg("old proposal received")

		return false, err
	}

	return true, nil
}
