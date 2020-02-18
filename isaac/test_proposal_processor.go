// +build test

package isaac

import "github.com/spikeekips/mitum/valuehash"

type DummyProposalProcessor struct {
	returnBlock Block
	err         error
}

func NewDummyProposalProcessor(returnBlock Block, err error) DummyProposalProcessor {
	return DummyProposalProcessor{returnBlock: returnBlock, err: err}
}

func (dp DummyProposalProcessor) Process(valuehash.Hash /* Proposal.Hash() */, []byte) (BlockStorage, error) {
	return &DummyBlockStorage{block: dp.returnBlock}, dp.err
}
