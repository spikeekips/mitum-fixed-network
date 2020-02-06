package mitum

type ProposalProcessor interface {
	Process(Proposal) (Block, error)
}
