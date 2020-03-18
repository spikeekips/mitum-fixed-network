package isaac

import (
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/operation"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/util"
)

type BallotChecker struct {
	*logging.Logging
	suffrage   Suffrage
	localstate *Localstate
	ballot     Ballot
}

func NewBallotChecker(ballot Ballot, localstate *Localstate, suffrage Suffrage) *BallotChecker {
	return &BallotChecker{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "ballot-checker")
		}),
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

func (bc *BallotChecker) CheckVoteproof() (bool, error) {
	var voteproof Voteproof
	switch t := bc.ballot.(type) {
	case INITBallot:
		voteproof = t.Voteproof()
	case ACCEPTBallot:
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

func IsValidBallot(ballot Ballot, b []byte) error {
	if err := seal.IsValidSeal(ballot, b); err != nil {
		return err
	}

	return operation.IsValidEmbededFact(ballot.Signer(), ballot, b)
}
