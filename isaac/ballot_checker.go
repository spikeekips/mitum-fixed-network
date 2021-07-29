package isaac

import (
	"context"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
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

// IsFromLocal filters ballots from local thru network; whether it is from the
// other node, which has same node address
func (bc *BallotChecker) IsFromLocal() (bool, error) {
	if bc.nodepool.LocalNode().Address().Equal(bc.ballot.Node()) {
		return false, nil
	}

	return true, nil
}

// InTimespan checks whether ballot is signed at a given interval,
// policy.TimespanValidBallot().
func (bc *BallotChecker) InTimespan() (bool, error) {
	if _, ok := bc.ballot.(ballot.Proposal); ok { // NOTE old signed proposal also can be correct
		return true, nil
	}

	if s := bc.ballot.SignedAt(); s.After(localtime.Now().Add(bc.policy.TimespanValidBallot())) {
		return false, xerrors.Errorf("too new ballot")
	} else if s.Before(localtime.Now().Add(bc.policy.TimespanValidBallot() * -1)) {
		return false, xerrors.Errorf("too old ballot")
	}

	return true, nil
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
	err := CheckBallotSigning(bc.ballot, bc.nodepool)
	return err == nil, err
}

func (bc *BallotChecker) IsFromAliveNode() (bool, error) {
	if _, ok := bc.ballot.(ballot.Proposal); ok {
		return true, nil
	}

	switch _, ch, found := bc.nodepool.Node(bc.ballot.Node()); {
	case !found:
		return false, xerrors.Errorf("unknown node, %q", bc.ballot.Node())
	case ch == nil:
		return false, xerrors.Errorf("from dead node, %q", bc.ballot.Node())
	}

	return true, nil
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
	i, ok := bc.ballot.(ballot.ACCEPT)
	if !ok {
		return true, nil
	}
	ph := i.Proposal()

	var proposal ballot.Proposal
	if i, found, err := bc.database.Seal(ph); err != nil {
		return false, err
	} else if found {
		j, ok := i.(ballot.Proposal)
		if !ok {
			return false, xerrors.Errorf("not proposal in accept ballot, %T", i)
		}
		proposal = j
	}

	if proposal == nil { // NOTE if not found, request proposal from node of ballot
		i, err := bc.requestProposalFromNodes(ph)
		if err != nil {
			return false, err
		}
		proposal = i
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
	i, ok := bc.ballot.(base.Voteproofer)
	if !ok {
		return true, nil
	}
	voteproof := i.Voteproof()

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

func (bc *BallotChecker) requestProposalFromNodes(h valuehash.Hash) (ballot.Proposal, error) {
	sem := semaphore.NewWeighted(100)
	eg, ctx := errgroup.WithContext(context.Background())

	bc.nodepool.TraverseAliveRemotes(func(no base.Node, ch network.Channel) bool {
		if !bc.suffrage.IsInside(no.Address()) {
			return true
		}

		if err := sem.Acquire(ctx, 1); err != nil {
			return false
		}

		eg.Go(func() error {
			return func(_ base.Node, ch network.Channel) error {
				defer sem.Release(1)

				pr, err := bc.requestProposal(ch, h)
				if err != nil {
					return nil // nolint:nilerr
				}

				return util.NewDataContainerError(pr)
			}(no, ch)
		})

		return true
	})

	if err := sem.Acquire(ctx, 100); err != nil {
		if !xerrors.Is(err, context.Canceled) {
			return nil, xerrors.Errorf("failed to request proposal: %w", err)
		}
	}

	err := eg.Wait()

	var dc util.DataContainerError
	if !xerrors.As(err, &dc) {
		return nil, xerrors.Errorf("failed to request proposal, %v", h)
	}

	return dc.Data().(ballot.Proposal), nil
}

func (bc *BallotChecker) requestProposal(ch network.Channel, h valuehash.Hash) (ballot.Proposal, error) {
	proposal, err := RequestProposal(ch, h)
	if err != nil {
		return nil, err
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
	node, _, found := nodepool.Node(blt.Node())
	if !found {
		return xerrors.Errorf("node not found")
	}

	if !blt.Signer().Equal(node.Publickey()) {
		return xerrors.Errorf("publickey not matched")
	}

	return nil
}

func RequestProposal(ch network.Channel, h valuehash.Hash) (ballot.Proposal, error) {
	if r, err := ch.Seals(context.TODO(), []valuehash.Hash{h}); err != nil {
		return nil, err
	} else if len(r) < 1 {
		return nil, xerrors.Errorf("no Proposal found, %v", h.String())
	} else if pr, ok := r[0].(ballot.Proposal); !ok {
		return nil, xerrors.Errorf(
			"request %v, but not ballot.Proposal, %T",
			h.String(),
			r[0],
		)
	} else {
		return pr, nil
	}
}
