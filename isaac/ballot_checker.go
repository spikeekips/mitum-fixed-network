package isaac

import (
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/util"
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

	if err := ballot.Signer().Verify(
		util.ConcatSlice([][]byte{ballot.FactHash().Bytes(), b}),
		ballot.FactSignature(),
	); err != nil {
		return err
	}

	return nil
}
