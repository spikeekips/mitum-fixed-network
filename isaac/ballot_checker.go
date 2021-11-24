package isaac

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
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
	ballot   base.Ballot
	fact     base.BallotFact
	factSign base.BallotFactSign
	lvp      base.Voteproof
}

func NewBallotChecker(
	blt base.Ballot,
	db storage.Database,
	policy *LocalPolicy,
	suffrage base.Suffrage,
	nodepool *network.Nodepool,
	lastVoteproof base.Voteproof,
) *BallotChecker {
	return &BallotChecker{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "ballot-checker")
		}),
		database: db,
		policy:   policy,
		suffrage: suffrage,
		nodepool: nodepool,
		ballot:   blt,
		fact:     blt.RawFact(),
		factSign: blt.FactSign(),
		lvp:      lastVoteproof,
	}
}

// IsFromLocal filters ballots from local thru network; whether it is from the
// other node, which has same node address
func (bc *BallotChecker) IsFromLocal() (bool, error) {
	if bc.nodepool.LocalNode().Address().Equal(bc.factSign.Node()) {
		return false, nil
	}

	return true, nil
}

// InTimespan checks whether ballot is signed at a given interval,
// policy.TimespanValidBallot().
func (bc *BallotChecker) InTimespan() (bool, error) {
	if bc.fact.Stage() == base.StageProposal { // NOTE proposal should be resigned except fact
		if !localtime.WithinNow(bc.ballot.SignedAt(), bc.policy.TimespanValidBallot()) {
			return false, errors.Errorf("too old or new proposal")
		}

		return true, nil
	}

	if !localtime.WithinNow(bc.ballot.FactSign().SignedAt(), bc.policy.TimespanValidBallot()) {
		return false, errors.Errorf("too old or new ballot")
	}

	return true, nil
}

// InSuffrage checks BallotFactSign.Node() is inside suffrage
func (bc *BallotChecker) InSuffrage() (bool, error) {
	if !bc.suffrage.IsInside(bc.factSign.Node()) {
		return false, nil
	}

	return true, nil
}

// CheckSigning checks node signed by it's valid key.
func (bc *BallotChecker) CheckSigning() (bool, error) {
	err := CheckBallotSigningNode(bc.ballot.FactSign(), bc.nodepool)
	return err == nil, err
}

func (bc *BallotChecker) IsFromAliveNode() (bool, error) {
	if bc.fact.Stage() == base.StageProposal {
		return true, nil
	}

	switch _, ch, found := bc.nodepool.Node(bc.factSign.Node()); {
	case !found:
		return false, errors.Errorf("unknown node, %q", bc.factSign.Node())
	case ch == nil:
		return false, errors.Errorf("from dead node, %q", bc.factSign.Node())
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

	bh := bc.fact.Height()
	lh := bc.lvp.Height()
	br := bc.fact.Round()
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
	i, ok := bc.fact.(base.ACCEPTBallotFact)
	if !ok {
		return true, nil
	}
	h := i.Proposal()

	var fact base.ProposalFact
	if i, found, err := bc.database.Proposal(h); err != nil {
		return false, err
	} else if found {
		fact = i.Fact()
	}

	if fact == nil { // NOTE if not found, request proposal from node of ballot
		i, err := bc.requestProposalFromNodes(h)
		if err != nil {
			return false, err
		}
		fact = i
	}

	if bc.fact.Height() != fact.Height() {
		return false, errors.Errorf("proposal in ACCEPTBallot is invalid; different height, ballot=%v proposal=%v",
			bc.fact.Height(), fact.Height())
	} else if bc.fact.Round() != fact.Round() {
		return false, errors.Errorf("proposal in ACCEPTBallot is invalid; different round, ballot=%v proposal=%v",
			bc.fact.Round(), fact.Round())
	}

	return true, nil
}

func (bc *BallotChecker) CheckVoteproof() (bool, error) {
	i, ok := bc.ballot.(interface{ BaseVoteproof() base.Voteproof })
	if !ok {
		return true, nil
	}
	voteproof := i.BaseVoteproof()

	vc := NewVoteProofChecker(voteproof, bc.policy, bc.suffrage)
	_ = vc.SetLogging(bc.Logging)

	if err := util.NewChecker("ballot-voteproof-checker", []util.CheckerFunc{
		vc.IsValid,
		vc.NodeIsInSuffrage,
		vc.CheckThreshold,
	}).Check(); err != nil {
		return false, err
	}

	return true, nil
}

func (bc *BallotChecker) requestProposalFromNodes(h valuehash.Hash) (base.ProposalFact, error) {
	wk := util.NewErrgroupWorker(context.Background(), 100)
	defer wk.Close()

	go func() {
		defer wk.Done()

		bc.nodepool.TraverseAliveRemotes(func(no base.Node, ch network.Channel) bool {
			if !bc.suffrage.IsInside(no.Address()) {
				return true
			}

			if err := wk.NewJob(func(ctx context.Context, _ uint64) error {
				return func(_ base.Node, ch network.Channel) error {
					pr, err := bc.requestProposal(context.Background(), ch, h)
					if err != nil {
						return nil // nolint:nilerr
					}

					return util.NewDataContainerError(pr)
				}(no, ch)
			}); err != nil {
				bc.Log().Error().Err(err).Msg("failed to NewJob for requesting Proposal")

				return false
			}

			return true
		})
	}()

	var dc util.DataContainerError
	if err := wk.Wait(); !errors.As(err, &dc) {
		if err != nil {
			return nil, fmt.Errorf("failed to request proposal, %v: %w", h, err)
		}

		return nil, errors.Errorf("failed to request proposal, %v", h)
	}

	return dc.Data().(base.Proposal).Fact(), nil
}

func (bc *BallotChecker) requestProposal(
	ctx context.Context, ch network.Channel, h valuehash.Hash,
) (base.Proposal, error) {
	proposal, err := ch.Proposal(ctx, h)
	switch {
	case err != nil:
		return nil, err
	case proposal == nil:
		return nil, util.NotFoundError
	}

	if err = proposal.IsValid(bc.policy.NetworkID()); err != nil {
		return nil, err
	}

	pvc, err := NewProposalValidationChecker(bc.database, bc.suffrage, bc.nodepool, proposal, nil)
	if err != nil {
		return nil, err
	}

	if err := util.NewChecker("proposal-checker", []util.CheckerFunc{
		pvc.IsKnown,
		pvc.CheckSigning,
		pvc.SaveProposal,
	}).Check(); err != nil {
		switch {
		case errors.Is(err, util.IgnoreError):
		case errors.Is(err, KnownSealError):
		default:
			return nil, err
		}
	}

	return proposal, nil
}

func CheckBallotSigningNode(fs base.BallotFactSign, nodepool *network.Nodepool) error {
	node, _, found := nodepool.Node(fs.Node())
	if !found {
		return errors.Errorf("node not found")
	}

	if !fs.Signer().Equal(node.Publickey()) {
		return errors.Errorf("publickey not matched")
	}

	return nil
}
