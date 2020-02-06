package isaac

type DummyProposalProcessor struct {
	returnBlock Block
	err         error
}

func NewDummyProposalProcessor(returnBlock Block, err error) DummyProposalProcessor {
	return DummyProposalProcessor{returnBlock: returnBlock, err: err}
}

func (dp DummyProposalProcessor) Process(Proposal) (Block, error) {
	return dp.returnBlock, dp.err
}
