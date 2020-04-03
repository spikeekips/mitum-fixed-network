// +build test

package isaac

import "github.com/spikeekips/mitum/valuehash"

type DummyProposalProcessor struct {
	returnBlock BlockUpdater
	err         error
	processed   map[valuehash.Hash]bool
	completed   map[valuehash.Hash]bool
}

func NewDummyProposalProcessor(returnBlock BlockUpdater, err error) *DummyProposalProcessor {
	return &DummyProposalProcessor{
		returnBlock: returnBlock,
		err:         err,
		processed:   map[valuehash.Hash]bool{},
		completed:   map[valuehash.Hash]bool{},
	}
}

func (dp *DummyProposalProcessor) IsProcessed(h valuehash.Hash) bool {
	return dp.processed[h]
}

func (dp *DummyProposalProcessor) ProcessINIT(h valuehash.Hash, initVoteproof Voteproof) (Block, error) {
	dp.processed[h] = true

	dp.returnBlock = dp.returnBlock.SetINITVoteproof(initVoteproof)
	return dp.returnBlock, dp.err
}

func (dp *DummyProposalProcessor) ProcessACCEPT(h valuehash.Hash, acceptVoteproof Voteproof) (BlockStorage, error) {
	dp.completed[h] = true

	dp.returnBlock = dp.returnBlock.SetACCEPTVoteproof(acceptVoteproof)
	return &DummyBlockStorage{block: dp.returnBlock}, dp.err
}
