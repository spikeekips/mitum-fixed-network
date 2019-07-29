package isaac

type ProposalValidator interface {
	NewBlock(Proposal) (Block, error)
}
