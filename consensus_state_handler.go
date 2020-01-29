package mitum

import "github.com/spikeekips/mitum/seal"

type ConsensusStateHandler interface {
	State() ConsensusState
	Activate() error
	Deactivate() error
	// NewSeal receives Seal.
	NewSeal(seal.Seal) error
	// NewVoteResult receives the finished VoteResult.
	NewVoteResult(VoteResult) error
}
