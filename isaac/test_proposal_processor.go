// +build test

package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/storage"
)

type DummyProposalProcessor struct {
	returnBlock block.BlockUpdater
	err         error
	processed   map[valuehash.Hash]bool
	completed   map[valuehash.Hash]bool
	bs          map[valuehash.Hash]*storage.DummyBlockStorage
}

func NewDummyProposalProcessor(returnBlock block.BlockUpdater, err error) *DummyProposalProcessor {
	return &DummyProposalProcessor{
		returnBlock: returnBlock,
		err:         err,
		processed:   map[valuehash.Hash]bool{},
		completed:   map[valuehash.Hash]bool{},
		bs:          map[valuehash.Hash]*storage.DummyBlockStorage{},
	}
}

func (dp *DummyProposalProcessor) SetReturnBlock(blk block.BlockUpdater) {
	dp.returnBlock = blk
}

func (dp *DummyProposalProcessor) SetError(err error) {
	dp.err = err
}

func (dp *DummyProposalProcessor) IsProcessed(h valuehash.Hash) bool {
	return dp.processed[h]
}

func (dp *DummyProposalProcessor) ProcessINIT(h valuehash.Hash, initVoteproof base.Voteproof) (block.Block, error) {
	dp.processed[h] = true

	dp.returnBlock = dp.returnBlock.SetINITVoteproof(initVoteproof)
	return dp.returnBlock, dp.err
}

func (dp *DummyProposalProcessor) ProcessACCEPT(
	h valuehash.Hash, acceptVoteproof base.Voteproof,
) (storage.BlockStorage, error) {
	dp.completed[h] = true

	dp.returnBlock = dp.returnBlock.SetACCEPTVoteproof(acceptVoteproof)

	if dp.err != nil {
		return nil, dp.err
	}

	bs := storage.NewDummyBlockStorage(dp.returnBlock, nil, nil)

	dp.bs[h] = bs

	return bs, nil
}

func (dp *DummyProposalProcessor) BlockStorages(h valuehash.Hash) *storage.DummyBlockStorage {
	return dp.bs[h]
}
