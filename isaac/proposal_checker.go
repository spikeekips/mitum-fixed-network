package isaac

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/logging"
)

type ProposalValidationChecker struct {
	*logging.Logging
	localstate *Localstate
	suffrage   base.Suffrage
	proposal   ballot.Proposal
}

func NewProposalValidationChecker(
	localstate *Localstate, suffrage base.Suffrage, proposal ballot.Proposal,
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

	_, err := pvc.localstate.Storage().Proposal(height, round)
	if err != nil {
		if !xerrors.Is(err, storage.NotFoundError) {
			return false, err
		}
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
	if err := pvc.localstate.Storage().NewProposal(pvc.proposal); err != nil {
		return false, err
	}

	return true, nil
}

func (pvc *ProposalValidationChecker) IsOld() (bool, error) {
	height := pvc.proposal.Height()
	round := pvc.proposal.Round()

	ivp := pvc.localstate.LastINITVoteproof()
	if height < ivp.Height() || round != ivp.Round() {
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
