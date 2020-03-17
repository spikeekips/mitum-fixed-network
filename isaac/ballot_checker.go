package isaac

import (
	"github.com/spikeekips/mitum/operation"
	"github.com/spikeekips/mitum/seal"
)

type BallotChecker struct {
	suffrage   Suffrage
	localstate *Localstate
	ballot     Ballot
}

func NewBallotChecker(ballot Ballot, localstate *Localstate, suffrage Suffrage) *BallotChecker {
	return &BallotChecker{
		suffrage:   suffrage,
		localstate: localstate,
		ballot:     ballot,
	}
}

// CheckIsInSuffrage checks Ballot.Node() is inside suffrage
func (bc *BallotChecker) CheckIsInSuffrage() (bool, error) {
	if !bc.suffrage.IsInside(bc.ballot.Node()) {
		return false, nil
	}

	return true, nil
}

// CheckWithLastBlock checks Ballot.Height() and Ballot.Round() with
// last Block.
// - If Height is same or lower than last, Ballot will be ignored.
func (bc *BallotChecker) CheckWithLastBlock() (bool, error) {
	block := bc.localstate.LastBlock()
	if bc.ballot.Height() <= block.Height() {
		return false, nil
	}

	return true, nil
}

func IsValidBallot(ballot Ballot, b []byte) error {
	if err := seal.IsValidSeal(ballot, b); err != nil {
		return err
	}

	return operation.IsValidEmbededFact(ballot.Signer(), ballot, b)
}
