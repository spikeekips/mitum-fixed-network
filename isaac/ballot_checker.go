package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

type BallotChecker struct {
	*logging.Logging
	suffrage   base.Suffrage
	localstate *Localstate
	blt        ballot.Ballot
}

func NewBallotChecker(blt ballot.Ballot, localstate *Localstate, suffrage base.Suffrage) *BallotChecker {
	return &BallotChecker{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "ballot-checker")
		}),
		suffrage:   suffrage,
		localstate: localstate,
		blt:        blt,
	}
}

// CheckIsInSuffrage checks Ballot.Node() is inside suffrage
func (bc *BallotChecker) CheckIsInSuffrage() (bool, error) {
	if !bc.suffrage.IsInside(bc.blt.Node()) {
		return false, nil
	}

	return true, nil
}

// CheckWithLastBlock checks Ballot.Height() and Ballot.Round() with
// last Block.
// - If Height is same or lower than last, Ballot will be ignored.
func (bc *BallotChecker) CheckWithLastBlock() (bool, error) {
	block := bc.localstate.LastBlock()
	if bc.blt.Height() <= block.Height() {
		return false, nil
	}

	return true, nil
}

func (bc *BallotChecker) CheckVoteproof() (bool, error) {
	var voteproof base.Voteproof
	switch t := bc.blt.(type) {
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
