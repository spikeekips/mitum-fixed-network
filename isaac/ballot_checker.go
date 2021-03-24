package isaac

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

type BallotChecker struct {
	*logging.Logging
	database storage.Database
	policy   *LocalPolicy
	suffrage base.Suffrage
	nodepool *network.Nodepool
	ballot   ballot.Ballot
	lvp      base.Voteproof
}

func NewBallotChecker(
	blt ballot.Ballot,
	st storage.Database,
	policy *LocalPolicy,
	suffrage base.Suffrage,
	nodepool *network.Nodepool,
	lastVoteproof base.Voteproof,
) *BallotChecker {
	return &BallotChecker{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "ballot-checker")
		}),
		database: st,
		policy:   policy,
		suffrage: suffrage,
		nodepool: nodepool,
		ballot:   blt,
		lvp:      lastVoteproof,
	}
}

// InSuffrage checks Ballot.Node() is inside suffrage
func (bc *BallotChecker) InSuffrage() (bool, error) {
	if !bc.suffrage.IsInside(bc.ballot.Node()) {
		return false, nil
	}

	return true, nil
}

// CheckSigning checks node signed by it's valid key.
func (bc *BallotChecker) CheckSigning() (bool, error) {
	if err := CheckBallotSigning(bc.ballot, bc.nodepool); err != nil {
		return false, err
	} else {
		return true, nil
	}
}

// CheckWithLastVoteproof checks Ballot.Height() and Ballot.Round() with
// last Block.
// - If Height is same or lower than last, Ballot will be ignored.
func (bc *BallotChecker) CheckWithLastVoteproof() (bool, error) {
	if bc.lvp == nil {
		return true, nil
	}

	bh := bc.ballot.Height()
	lh := bc.lvp.Height()
	br := bc.ballot.Round()
	lr := bc.lvp.Round()

	switch {
	case bh < lh:
		return false, nil
	case bh > lh:
		return true, nil
	case br <= lr:
		return false, nil
	default:
		return true, nil
	}
}

// CheckProposalInACCEPTBallot checks ACCEPT ballot should have valid proposal.
func (bc *BallotChecker) CheckProposalInACCEPTBallot() (bool, error) {
	var ph valuehash.Hash
	if i, ok := bc.ballot.(ballot.ACCEPTBallot); !ok {
		return true, nil
	} else {
		ph = i.Proposal()
	}

	var proposal ballot.Proposal
	if i, found, err := bc.database.Seal(ph); err != nil {
		return false, err
	} else if found {
		if j, ok := i.(ballot.Proposal); !ok {
			return false, xerrors.Errorf("not proposal in accept ballot, %T", i)
		} else {
			proposal = j
		}
	}

	if proposal == nil { // NOTE if not found, request proposal from node of ballot
		if i, err := bc.requestProposal(bc.ballot.Node(), ph); err != nil {
			return false, err
		} else {
			proposal = i
		}
	}

	if bc.ballot.Height() != proposal.Height() {
		return false, xerrors.Errorf("proposal in ACCEPTBallot is invalid; different height, ballot=%v proposal=%v",
			bc.ballot.Height(), proposal.Height())
	} else if bc.ballot.Round() != proposal.Round() {
		return false, xerrors.Errorf("proposal in ACCEPTBallot is invalid; different round, ballot=%v proposal=%v",
			bc.ballot.Round(), proposal.Round())
	}

	return true, nil
}

func (bc *BallotChecker) CheckVoteproof() (bool, error) {
	var voteproof base.Voteproof
	if i, ok := bc.ballot.(base.Voteproofer); !ok {
		return true, nil
	} else {
		voteproof = i.Voteproof()
	}

	vc := NewVoteProofChecker(voteproof, bc.policy, bc.suffrage)
	_ = vc.SetLogger(bc.Log())

	if err := util.NewChecker("ballot-voteproof-checker", []util.CheckerFunc{
		vc.IsValid,
		vc.NodeIsInSuffrage,
		vc.CheckThreshold,
	}).Check(); err != nil {
		return false, err
	}

	return true, nil
}

func (bc *BallotChecker) requestProposal(address base.Address, h valuehash.Hash) (ballot.Proposal, error) {
	var proposal ballot.Proposal
	if n, found := bc.nodepool.Node(address); !found {
		return nil, xerrors.Errorf("unknown node of ballot; %v", address)
	} else if i, err := RequestProposal(n, h); err != nil {
		return nil, err
	} else {
		proposal = i
	}

	sealChecker := NewSealChecker(proposal, bc.database, bc.policy, nil)
	if err := util.NewChecker("proposal-seal-checker", []util.CheckerFunc{sealChecker.IsValid}).Check(); err != nil {
		return nil, err
	}

	ballotChecker := NewBallotChecker(proposal, bc.database, bc.policy, bc.suffrage, bc.nodepool, bc.lvp)
	if err := util.NewChecker("proposal-ballot-checker", []util.CheckerFunc{
		ballotChecker.InSuffrage,
		ballotChecker.CheckVoteproof,
	}).Check(); err != nil {
		if !xerrors.Is(err, util.IgnoreError) {
			return nil, err
		}
	}

	pvc := NewProposalValidationChecker(bc.database, bc.suffrage, bc.nodepool, proposal, nil)
	if err := util.NewChecker("proposal-checker", []util.CheckerFunc{
		pvc.IsKnown,
		pvc.CheckSigning,
		pvc.SaveProposal,
	}).Check(); err != nil {
		switch {
		case xerrors.Is(err, util.IgnoreError):
		case xerrors.Is(err, KnownSealError):
		default:
			return nil, err
		}
	}

	return proposal, nil
}

func CheckBallotSigning(blt ballot.Ballot, nodepool *network.Nodepool) error {
	var node base.Node
	if n, found := nodepool.Node(blt.Node()); !found {
		return xerrors.Errorf("node not found")
	} else {
		node = n
	}

	if !blt.Signer().Equal(node.Publickey()) {
		return xerrors.Errorf("publickey not matched")
	}

	return nil
}

func RequestProposal(node network.Node, h valuehash.Hash) (ballot.Proposal, error) {
	if r, err := node.Channel().Seals(context.TODO(), []valuehash.Hash{h}); err != nil {
		return nil, err
	} else if len(r) < 1 {
		return nil, xerrors.Errorf(
			"failed to receive Proposal, %v from %s",
			h.String(),
			node.Address(),
		)
	} else if pr, ok := r[0].(ballot.Proposal); !ok {
		return nil, xerrors.Errorf(
			"failed to receive Proposal, %v from %s; not ballot.Proposal, %T",
			h.String(),
			node.Address(),
			r[0],
		)
	} else {
		return pr, nil
	}
}
