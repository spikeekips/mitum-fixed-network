package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

type ProposalProcessor interface {
	util.Initializer
	IsProcessed(valuehash.Hash /* proposal.Hash() */) bool
	ProcessINIT(valuehash.Hash /* Proposal.Hash() */, base.Voteproof /* INIT Voteproof */) (block.Block, error)
	ProcessACCEPT(
		valuehash.Hash /* Proposal.Hash() */, base.Voteproof, /* ACCEPT Voteproof */
	) (storage.BlockStorage, error)
	Done(valuehash.Hash /* Proposal.Hash() */) error
	Cancel() error
	AddOperationProcessor(hint.Hinter, OperationProcessor) (ProposalProcessor, error)
	States() map[string]interface{}
}

type OperationProcessor interface {
	New(*Statepool) OperationProcessor
	Process(state.StateProcessor) error
}

type defaultOperationProcessor struct {
	pool *Statepool
}

func (opp defaultOperationProcessor) New(pool *Statepool) OperationProcessor {
	return &defaultOperationProcessor{
		pool: pool,
	}
}

func (opp defaultOperationProcessor) Process(op state.StateProcessor) error {
	return op.Process(opp.pool.Get, opp.pool.Set)
}
