package mitum

import "github.com/spikeekips/mitum/seal"

type ConsensusStateHandler interface {
	State() ConsensusState
	SetStateChan(chan<- ConsensusStateChangeContext)
	Activate() error
	Deactivate() error
	// NewSeal receives Seal.
	NewSeal(seal.Seal) error
	// NewVoteProof receives the finished VoteProof.
	NewVoteProof(VoteProof) error
}

type ConsensusStateChangeContext struct {
	fromState ConsensusState
	toState   ConsensusState
	voteProof VoteProof
}

func (csc ConsensusStateChangeContext) From() ConsensusState {
	return csc.fromState
}

func (csc ConsensusStateChangeContext) To() ConsensusState {
	return csc.toState
}

func (csc ConsensusStateChangeContext) VoteProof() VoteProof {
	return csc.voteProof
}
