package isaac

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/util/logging"
)

type ProposalValidationChecker struct {
	*logging.Logging
	localstate    *Localstate
	suffrage      base.Suffrage
	proposal      ballot.Proposal
	initVoteproof base.Voteproof
}

func NewProposalValidationChecker(
	localstate *Localstate,
	suffrage base.Suffrage,
	proposal ballot.Proposal,
	initVoteproof base.Voteproof,
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
		localstate:    localstate,
		suffrage:      suffrage,
		proposal:      proposal,
		initVoteproof: initVoteproof,
	}
}

// IsKnown checks proposal is already received; if found, no nore checks.
func (pvc *ProposalValidationChecker) IsKnown() (bool, error) {
	height := pvc.proposal.Height()
	round := pvc.proposal.Round()

	if _, found, err := pvc.localstate.Storage().Proposal(height, round); err != nil {
		return false, err
	} else if found {
		return false, nil // NOTE the already saved will be passed
	}

	return true, nil
}

// CheckSigning checks node signed by it's valid key.
func (pvc *ProposalValidationChecker) CheckSigning() (bool, error) {
	var node base.Node
	if pvc.proposal.Node().Equal(pvc.localstate.Node().Address()) {
		node = pvc.localstate.Node()
	} else if n, found := pvc.localstate.Nodes().Node(pvc.proposal.Node()); !found {
		return false, xerrors.Errorf("node not found")
	} else {
		node = n
	}

	if !pvc.proposal.Signer().Equal(node.Publickey()) {
		return false, xerrors.Errorf("publickey not matched")
	}

	return true, nil
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
		Hinted("expected_proposer", pvc.suffrage.Acting(height, round).Proposer()).
		Send()

	pvc.Log().Error().Err(err).Msg("wrong proposer found")

	return false, err
}

func (pvc *ProposalValidationChecker) SaveProposal() (bool, error) {
	// NOTE befor saving proposal, check again for preventing duplication error
	height := pvc.proposal.Height()
	round := pvc.proposal.Round()

	if _, found, err := pvc.localstate.Storage().Proposal(height, round); err != nil {
		return false, err
	} else if found {
		return true, nil // NOTE the already saved will be passed
	}

	if err := pvc.localstate.Storage().NewProposal(pvc.proposal); err != nil {
		return false, xerrors.Errorf("failed to save proposal: %w", err)
	}

	return true, nil
}

func (pvc *ProposalValidationChecker) IsOldOrHigher() (bool, error) {
	height := pvc.proposal.Height()
	round := pvc.proposal.Round()

	if height < pvc.initVoteproof.Height() || round != pvc.initVoteproof.Round() {
		err := xerrors.Errorf("old Proposal received")
		pvc.Log().Error().Err(err).
			Dict("current", logging.Dict().
				Hinted("height", pvc.initVoteproof.Height()).
				Hinted("round", pvc.initVoteproof.Round()),
			).
			Msg("old proposal received")

		return false, err
	} else if height > pvc.initVoteproof.Height() {
		err := xerrors.Errorf("higher Proposal received")
		pvc.Log().Error().Err(err).
			Dict("current", logging.Dict().
				Hinted("height", pvc.initVoteproof.Height()).
				Hinted("round", pvc.initVoteproof.Round()),
			).
			Msg("higher proposal received")

		return false, err
	}

	return true, nil
}
