package mitum

type ConsensusStateHandler interface {
	State() ConsensusState
	Activate() error
	Deactivate() error
	// NewProposal receives new Proposal from proposer.
	NewProposal(Proposal) error
	// NewVoteResult receives the finished VoteResult.
	NewVoteResult(VoteResult) error
}
