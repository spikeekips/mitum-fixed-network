package isaac

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

type ProposalChecker struct {
	*logging.Logging
	database storage.Database
	suffrage base.Suffrage
	nodepool *network.Nodepool
	proposal ballot.Proposal
	livp     base.Voteproof
}

func NewProposalValidationChecker(
	st storage.Database,
	suffrage base.Suffrage,
	nodepool *network.Nodepool,
	proposal ballot.Proposal,
	lastINITVoteproof base.Voteproof,
) *ProposalChecker {
	return &ProposalChecker{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.
				Str("module", "proposal-validation-checker").
				Stringer("proposal", proposal.Hash()).
				Int64("proposal_height", proposal.Height().Int64()).
				Uint64("proposal_round", proposal.Round().Uint64()).
				Stringer("proposal_node", proposal.Node())
		}),
		database: st,
		suffrage: suffrage,
		nodepool: nodepool,
		proposal: proposal,
		livp:     lastINITVoteproof,
	}
}

// IsKnown checks proposal is already received; if found, no nore checks.
func (pvc *ProposalChecker) IsKnown() (bool, error) {
	height := pvc.proposal.Height()
	round := pvc.proposal.Round()

	if _, found, err := pvc.database.Proposal(height, round, pvc.proposal.Node()); err != nil {
		return false, err
	} else if found {
		return false, KnownSealError.Merge(util.FoundError.Errorf("proposal already in database"))
	}

	return true, nil
}

// CheckSigning checks node signed by it's valid key.
func (pvc *ProposalChecker) CheckSigning() (bool, error) {
	err := CheckBallotSigning(pvc.proposal, pvc.nodepool)
	return err == nil, err
}

func (pvc *ProposalChecker) IsProposer() (bool, error) {
	if err := CheckNodeIsProposer(
		pvc.proposal.Node(),
		pvc.suffrage,
		pvc.proposal.Height(),
		pvc.proposal.Round(),
	); err != nil {
		return false, err
	}

	return true, nil
}

func (pvc *ProposalChecker) SaveProposal() (bool, error) {
	switch err := pvc.database.NewProposal(pvc.proposal); {
	case err == nil:
		return true, nil
	case errors.Is(err, util.DuplicatedError):
		return true, nil
	default:
		return false, err
	}
}

func (pvc *ProposalChecker) IsOlder() (bool, error) {
	if pvc.livp == nil {
		return false, errors.Errorf("no last voteproof")
	}

	ph := pvc.proposal.Height()
	lh := pvc.livp.Height()
	pr := pvc.proposal.Round()
	lr := pvc.livp.Round()

	switch {
	case ph < lh:
		return false, errors.Errorf("lower proposal height than last voteproof: %v < %v", ph, lh)
	case ph == lh && pr < lr:
		return false, errors.Errorf(
			"same height, but lower proposal round than last voteproof: %v < %v", pr, lr)
	default:
		return true, nil
	}
}

func (pvc *ProposalChecker) IsWaiting() (bool, error) {
	if pvc.livp == nil {
		return false, errors.Errorf("no last voteproof")
	}

	ph := pvc.proposal.Height()
	lh := pvc.livp.Height()
	pr := pvc.proposal.Round()
	lr := pvc.livp.Round()

	switch {
	case ph != lh:
		return false, errors.Errorf("proposal height does not match with last voteproof: %v != %v", ph, lh)
	case pr != lr:
		return false, errors.Errorf(
			"proposal round does not match with last voteproof: %v != %v", pr, lr)
	default:
		return true, nil
	}
}

func CheckNodeIsProposer(node base.Address, suffrage base.Suffrage, height base.Height, round base.Round) error {
	acting, err := suffrage.Acting(height, round)
	if err != nil {
		return err
	}

	if node.Equal(acting.Proposer()) {
		return nil
	}

	return errors.Errorf("proposal has wrong proposer")
}
