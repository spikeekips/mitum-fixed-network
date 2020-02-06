package isaac

type ProposalProcessor interface {
	Process(Proposal) (Block, error)
}
