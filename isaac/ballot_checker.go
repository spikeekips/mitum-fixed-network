package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

type BallotChecker struct {
	*logging.Logging
	suffrage   base.Suffrage
	localstate *Localstate
	ballot     ballot.Ballot
}

func NewBallotChecker(blt ballot.Ballot, localstate *Localstate, suffrage base.Suffrage) *BallotChecker {
	return &BallotChecker{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "ballot-checker")
		}),
		suffrage:   suffrage,
		localstate: localstate,
		ballot:     blt,
	}
}

// CheckIsInSuffrage checks Ballot.Node() is inside suffrage
func (bc *BallotChecker) CheckIsInSuffrage() (bool, error) {
	if !bc.suffrage.IsInside(bc.ballot.Node()) {
		return false, nil
	}

	return true, nil
}

// CheckSigning checks node signed by it's valid key.
func (bc *BallotChecker) CheckSigning() (bool, error) {
	var node base.Node
	if bc.ballot.Node().Equal(bc.localstate.Node().Address()) {
		node = bc.localstate.Node()
	} else if n, found := bc.localstate.Nodes().Node(bc.ballot.Node()); !found {
		return false, xerrors.Errorf("node not found")
	} else {
		node = n
	}

	if !bc.ballot.Signer().Equal(node.Publickey()) {
		return false, xerrors.Errorf("publickey not matched")
	}

	return true, nil
}

// CheckWithLastBlock checks Ballot.Height() and Ballot.Round() with
// last Block.
// - If Height is same or lower than last, Ballot will be ignored.
func (bc *BallotChecker) CheckWithLastBlock() (bool, error) {
	if bc.ballot.Height() <= bc.localstate.LastBlock().Height() {
		return false, nil
	}

	return true, nil
}

// CheckProposal checks ACCEPT ballot should have valid proposal.
func (bc *BallotChecker) CheckProposal() (bool, error) {
	var ph valuehash.Hash
	switch t := bc.ballot.(type) {
	case ballot.ACCEPTBallot:
		ph = t.Proposal()
	default:
		return true, nil
	}

	var proposal ballot.Proposal
	if sl, err := bc.localstate.Storage().Seal(ph); err != nil {
		if !xerrors.Is(err, storage.NotFoundError) {
			return false, err
		} else if pr, err := bc.requestProposal(bc.ballot.Node(), ph); err != nil {
			// NOTE if not found, request proposal from node of ballot
			return false, err
		} else {
			proposal = pr
		}
	} else if pr, ok := sl.(ballot.Proposal); !ok {
		return false, xerrors.Errorf("seal is not Proposal: %T", sl)
	} else {
		proposal = pr
	}

	if err := proposal.IsValid(bc.localstate.Policy().NetworkID()); err != nil {
		return false, err
	} else {
		pvc := NewProposalValidationChecker(bc.localstate, bc.suffrage, proposal)
		if err := util.NewChecker("proposal-validation-checker", []util.CheckerFunc{
			pvc.IsKnown,
			pvc.CheckSigning,
			pvc.IsProposer,
			pvc.SaveProposal,
			// pvc.IsOld, // NOTE duplicated function with belows.
		}).Check(); err != nil {
			return false, err
		}
	}

	if bc.ballot.Height() != proposal.Height() {
		return false, xerrors.Errorf(
			"proposal in ACCEPTBallot is invalid; different height, ballot=%v proposal=%v",
			bc.ballot.Height(), proposal.Height(),
		)
	}

	if bc.ballot.Round() != proposal.Round() {
		return false, xerrors.Errorf(
			"proposal in ACCEPTBallot is invalid; different round, ballot=%v proposal=%v",
			bc.ballot.Round(), proposal.Round(),
		)
	}

	return true, nil
}

func (bc *BallotChecker) CheckVoteproof() (bool, error) {
	var voteproof base.Voteproof
	switch t := bc.ballot.(type) {
	case ballot.INITBallot:
		voteproof = t.Voteproof()
	case ballot.ACCEPTBallot:
		voteproof = t.Voteproof()
	default:
		return true, nil
	}

	vc := NewVoteProofChecker(voteproof, bc.localstate, bc.suffrage)
	_ = vc.SetLogger(bc.Log())

	if err := util.NewChecker("ballot-voteproof-checker", []util.CheckerFunc{
		vc.CheckIsValid,
		vc.CheckNodeIsInSuffrage,
		vc.CheckThreshold,
	}).Check(); err != nil {
		return false, err
	}

	return true, nil
}

func (bc *BallotChecker) requestProposal(address base.Address, h valuehash.Hash) (ballot.Proposal, error) {
	if n, found := bc.localstate.Nodes().Node(address); !found {
		return nil, xerrors.Errorf("unknown node of ballot; %v", address)
	} else if r, err := n.Channel().Seals([]valuehash.Hash{h}); err != nil {
		return nil, err
	} else if len(r) < 1 {
		return nil, xerrors.Errorf(
			"failed to receive Proposal, %v from %s",
			h.String(),
			address,
		)
	} else if pr, ok := r[0].(ballot.Proposal); !ok {
		return nil, xerrors.Errorf(
			"failed to receive Proposal, %v from %s; not ballot.Proposal, %T",
			h.String(),
			address,
			r[0],
		)
	} else {
		return pr, nil
	}
}
