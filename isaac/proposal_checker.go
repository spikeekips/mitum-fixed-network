package isaac

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
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
	proposal base.Proposal
	fact     base.ProposalFact
	factSign base.BallotFactSign
	livp     base.Voteproof
}

func NewProposalValidationChecker(
	db storage.Database,
	suffrage base.Suffrage,
	nodepool *network.Nodepool,
	proposal base.Proposal,
	lastINITVoteproof base.Voteproof,
) (*ProposalChecker, error) {
	fact := proposal.Fact()
	return &ProposalChecker{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.
				Str("module", "proposal-validation-checker").
				Dict("proposal", zerolog.Dict().
					Stringer("hash", fact.Hash()).
					Stringer("fact", fact.Hash()).
					Int64("height", fact.Height().Int64()).
					Uint64("round", fact.Round().Uint64()).
					Stringer("proposer", fact.Proposer()),
				)
		}),
		database: db,
		suffrage: suffrage,
		nodepool: nodepool,
		proposal: proposal,
		fact:     fact,
		factSign: proposal.FactSign(),
		livp:     lastINITVoteproof,
	}, nil
}

// IsKnown checks proposal is already received; if found, no nore checks.
func (pvc *ProposalChecker) IsKnown() (bool, error) {
	if _, found, err := pvc.database.Proposal(pvc.fact.Hash()); err != nil {
		return false, err
	} else if found {
		return false, KnownSealError.Merge(util.FoundError.Errorf("proposal already in database"))
	}

	return true, nil
}

// CheckSigning checks node signed by it's valid key.
func (pvc *ProposalChecker) CheckSigning() (bool, error) {
	err := CheckBallotSigningNode(pvc.factSign, pvc.nodepool)
	return err == nil, err
}

func (pvc *ProposalChecker) IsProposer() (bool, error) {
	if err := CheckNodeIsProposer(
		pvc.fact.Proposer(),
		pvc.suffrage,
		pvc.fact.Height(),
		pvc.fact.Round(),
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

	ph := pvc.fact.Height()
	lh := pvc.livp.Height()
	pr := pvc.fact.Round()
	lr := pvc.livp.Round()

	switch {
	case ph < lh:
		return false, errors.Errorf("lower proposal height than last voteproof: %v < %v", ph, lh)
	case ph == lh && pr < lr:
		return false, errors.Errorf(
			"same height, but lower proposal round than last voteproof: %v < %v", pr, lr)
	}

	switch m, found, err := pvc.database.LastManifest(); {
	case err != nil || !found:
	case ph <= m.Height():
		return false, errors.Errorf("lower proposal height than last manifest: %v < %v", ph, m.Height())
	}

	return true, nil
}

func (pvc *ProposalChecker) IsWaiting() (bool, error) {
	if pvc.livp == nil {
		return false, errors.Errorf("no last voteproof")
	}

	ph := pvc.fact.Height()
	lh := pvc.livp.Height()
	pr := pvc.fact.Round()
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

func CheckNodeIsProposer(n base.Address, suffrage base.Suffrage, height base.Height, round base.Round) error {
	acting, err := suffrage.Acting(height, round)
	if err != nil {
		return err
	}

	if n.Equal(acting.Proposer()) {
		return nil
	}

	return errors.Errorf("proposal has wrong proposer")
}
