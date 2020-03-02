package isaac

import (
	"github.com/spikeekips/mitum/seal"
)

type BallotValidationChecker struct {
}

// (TODO) validate Ballot

type BallotChecker struct {
}

// (TODO) Ballot.Node() is inside suffrage
// (TODO) Ballot.Height() is equal or higher than LastINITVoteproof.
// (TODO) Ballot.Round() is equal or higher than LastINITVoteproof.

func IsValidBallot(ballot Ballot, b []byte) error {
	if err := seal.IsValidSeal(ballot, b); err != nil {
		return err
	}

	return IsValidEmbededFact(ballot.Signer(), ballot, b)
}
