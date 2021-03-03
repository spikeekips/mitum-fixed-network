package isaac

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/logging"
)

type ProposalValidationChecker struct {
	*logging.Logging
	local    *network.LocalNode
	storage  storage.Storage
	suffrage base.Suffrage
	nodepool *network.Nodepool
	proposal ballot.Proposal
	livp     base.Voteproof
}

func NewProposalValidationChecker(
	local *network.LocalNode,
	st storage.Storage,
	suffrage base.Suffrage,
	nodepool *network.Nodepool,
	proposal ballot.Proposal,
	lastINITVoteproof base.Voteproof,
) *ProposalValidationChecker {
	return &ProposalValidationChecker{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.
				Str("module", "proposal-validation-checker").
				Hinted("proposal", proposal.Hash()).
				Hinted("proposal_height", proposal.Height()).
				Hinted("proposal_round", proposal.Round()).
				Hinted("proposal_node", proposal.Node())
		}),
		local:    local,
		storage:  st,
		suffrage: suffrage,
		nodepool: nodepool,
		proposal: proposal,
		livp:     lastINITVoteproof,
	}
}

// IsKnown checks proposal is already received; if found, no nore checks.
func (pvc *ProposalValidationChecker) IsKnown() (bool, error) {
	height := pvc.proposal.Height()
	round := pvc.proposal.Round()

	if _, found, err := pvc.storage.Proposal(height, round, pvc.proposal.Node()); err != nil {
		return false, err
	} else if found {
		return false, KnownSealError.Wrap(storage.FoundError.Errorf("proposal already in storage"))
	}

	return true, nil
}

// CheckSigning checks node signed by it's valid key.
func (pvc *ProposalValidationChecker) CheckSigning() (bool, error) {
	if err := CheckBallotSigning(pvc.proposal, pvc.local, pvc.nodepool); err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func (pvc *ProposalValidationChecker) IsProposer() (bool, error) {
	if err := CheckNodeIsProposer(
		pvc.proposal.Node(),
		pvc.suffrage,
		pvc.proposal.Height(),
		pvc.proposal.Round(),
	); err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func (pvc *ProposalValidationChecker) SaveProposal() (bool, error) {
	switch err := pvc.storage.NewProposal(pvc.proposal); {
	case err == nil:
		return true, nil
	case xerrors.Is(err, storage.DuplicatedError):
		return true, nil
	default:
		return false, err
	}
}

func (pvc *ProposalValidationChecker) IsOlder() (bool, error) {
	if pvc.livp == nil {
		return false, xerrors.Errorf("no last voteproof")
	}

	ph := pvc.proposal.Height()
	lh := pvc.livp.Height()
	pr := pvc.proposal.Round()
	lr := pvc.livp.Round()

	switch {
	case ph < lh:
		return false, xerrors.Errorf("lower proposal height than last voteproof: %v < %v", ph, lh)
	case ph == lh && pr < lr:
		return false, xerrors.Errorf(
			"same height, but lower proposal round than last voteproof: %v < %v", pr, lr)
	default:
		return true, nil
	}
}

func (pvc *ProposalValidationChecker) IsWaiting() (bool, error) {
	if pvc.livp == nil {
		return false, xerrors.Errorf("no last voteproof")
	}

	ph := pvc.proposal.Height()
	lh := pvc.livp.Height()
	pr := pvc.proposal.Round()
	lr := pvc.livp.Round()

	switch {
	case ph != lh:
		return false, xerrors.Errorf("proposal height does not match with last voteproof: %v != %v", ph, lh)
	case pr != lr:
		return false, xerrors.Errorf(
			"proposal round does not match with last voteproof: %v != %v", pr, lr)
	default:
		return true, nil
	}
}

func CheckNodeIsProposer(node base.Address, suffrage base.Suffrage, height base.Height, round base.Round) error {
	var acting base.ActingSuffrage
	if i, err := suffrage.Acting(height, round); err != nil {
		return err
	} else {
		acting = i
	}

	if node.Equal(acting.Proposer()) {
		return nil
	} else {
		return xerrors.Errorf("proposal has wrong proposer")
	}
}
