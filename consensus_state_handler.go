package mitum

import "github.com/spikeekips/mitum/seal"

type ConsensusStateHandler interface {
	State() ConsensusState
	Activate() error
	Deactivate() error
	// NewSeal receives Seal.
	NewSeal(seal.Seal) error
	// NewVoteProof receives the finished VoteProof.
	NewVoteProof(VoteProof) error
}
