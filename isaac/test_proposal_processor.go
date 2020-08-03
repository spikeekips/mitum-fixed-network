// +build test

package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

type DummyProposalProcessor struct {
	returnBlock block.BlockUpdater
	err         error
	processed   map[string]bool
	completed   map[string]bool
	bs          map[string]*storage.DummyBlockStorage
}

func NewDummyProposalProcessor(returnBlock block.BlockUpdater, err error) *DummyProposalProcessor {
	return &DummyProposalProcessor{
		returnBlock: returnBlock,
		err:         err,
		processed:   map[string]bool{},
		completed:   map[string]bool{},
		bs:          map[string]*storage.DummyBlockStorage{},
	}
}

func (dp *DummyProposalProcessor) Initialize() error {
	return nil
}

func (dp *DummyProposalProcessor) SetReturnBlock(blk block.BlockUpdater) {
	dp.returnBlock = blk
}

func (dp *DummyProposalProcessor) SetError(err error) {
	dp.err = err
}

func (dp *DummyProposalProcessor) IsProcessed(h valuehash.Hash) bool {
	return dp.processed[h.String()]
}

func (dp *DummyProposalProcessor) ProcessINIT(h valuehash.Hash, initVoteproof base.Voteproof) (block.Block, error) {
	dp.processed[h.String()] = true

	dp.returnBlock = dp.returnBlock.SetINITVoteproof(initVoteproof)
	return dp.returnBlock, dp.err
}

func (dp *DummyProposalProcessor) ProcessACCEPT(
	h valuehash.Hash, acceptVoteproof base.Voteproof,
) (storage.BlockStorage, error) {
	dp.completed[h.String()] = true

	dp.returnBlock = dp.returnBlock.SetACCEPTVoteproof(acceptVoteproof)

	if dp.err != nil {
		return nil, dp.err
	}

	bs := storage.NewDummyBlockStorage(dp.returnBlock, nil, nil)

	dp.bs[h.String()] = bs

	return bs, nil
}

func (dp *DummyProposalProcessor) Done(valuehash.Hash) error {
	return nil
}

func (dp *DummyProposalProcessor) BlockStorages(h valuehash.Hash) *storage.DummyBlockStorage {
	return dp.bs[h.String()]
}

func (dp *DummyProposalProcessor) AddOperationProcessor(hint.Hinter, OperationProcessor) (ProposalProcessor, error) {
	return dp, nil
}
