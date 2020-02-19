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

func (dp DummyProposalProcessor) ProcessINIT(_ valuehash.Hash, initVoteproof Voteproof, _ []byte) (Block, error) {
	dp.returnBlock = dp.returnBlock.SetINITVoteproof(initVoteproof)
	return dp.returnBlock, dp.err
}

func (dp DummyProposalProcessor) ProcessACCEPT(
	_ valuehash.Hash, acceptVoteproof Voteproof, _ []byte,
) (BlockStorage, error) {
	dp.returnBlock = dp.returnBlock.SetACCEPTVoteproof(acceptVoteproof)
	return &DummyBlockStorage{block: dp.returnBlock}, dp.err
}
