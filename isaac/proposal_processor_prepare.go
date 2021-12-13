package isaac

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/tree"
	"github.com/spikeekips/mitum/util/valuehash"
)

var blockDataMapContextKey util.ContextKey = "blockdata_map"

func (pp *DefaultProcessor) Prepare(ctx context.Context) (block.Block, error) {
	pp.Lock()
	defer pp.Unlock()

	prepareCtx, prepareCancel := context.WithCancel(ctx)
	pp.prepareCtx = prepareCtx
	pp.prepareCancel = prepareCancel

	started := time.Now()
	defer func() {
		_ = pp.setStatic("processor_prepare_elapsed", time.Since(started))
	}()

	if err := pp.resetPrepare(); err != nil {
		return nil, err
	}

	pp.setState(prprocessor.Preparing)

	if err := pp.prepare(prepareCtx); err != nil {
		pp.setState(prprocessor.PrepareFailed)

		if err0 := pp.resetPrepare(); err0 != nil {
			return nil, err0
		}

		return nil, err
	}
	pp.setState(prprocessor.Prepared)

	return pp.blk, nil
}

func (pp *DefaultProcessor) prepare(ctx context.Context) error {
	if pp.prePrepareHook != nil {
		if err := pp.prePrepareHook(ctx); err != nil {
			return err
		}
	}

	for _, f := range []func(context.Context) error{
		pp.prepareBlockDataSession,
		pp.prepareOperations,
		pp.process,
		pp.prepareBlock,
		pp.prepareDatabaseSession,
	} {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := f(ctx); err != nil {
				pp.Log().Error().Err(err).Msg("failed to prepare")

				return err
			}
		}
	}

	if pp.postPrepareHook != nil {
		if err := pp.postPrepareHook(ctx); err != nil {
			pp.Log().Error().Err(err).Msg("failed postPrepareHook")

			return err
		}
	}

	pp.Log().Debug().Msg("prepared")

	return nil
}

func (pp *DefaultProcessor) prepareOperations(ctx context.Context) error {
	ev := pp.Log().Debug().Int("operations", len(pp.Fact().Operations()))

	opsh := pp.Fact().Operations()

	started := time.Now()
	defer func() {
		_ = pp.setStatic("processor_prepare_operations_elapsed", time.Since(started)).
			setStatic("processor_prepare_operations_operation_hashes", len(opsh)).
			setStatic("processor_prepare_operations_operations", len(pp.operations))
	}()

	if len(opsh) < 1 {
		return nil
	}

	se := NewOperationsExtractor(pp.nodepool.LocalNode().Address(), pp.Fact().Proposer(), pp.database, pp.nodepool, opsh)
	_ = se.SetLogging(pp.Logging)

	ops, err := se.Extract(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to extract seals")
	}

	ev.Int("operations", len(ops)).Msg("operations extracted from seals of proposal")

	pp.operations = ops

	return nil
}

func (pp *DefaultProcessor) process(ctx context.Context) error {
	started := time.Now()
	defer func() {
		_ = pp.setStatic("processor_process_elapsed", time.Since(started))
	}()

	if err := pp.processBlockDataSessionAddOperations(); err != nil {
		return err
	}

	if err := pp.processOperations(ctx); err != nil {
		return err
	}

	if err := pp.processBlockDataSessionSetOperationsTree(); err != nil {
		return err
	} else if err := pp.processBlockDataSessionSetStatesTree(); err != nil {
		return err
	}

	return pp.processBlockDataSessionSetStates()
}

func (pp *DefaultProcessor) processBlockDataSessionAddOperations() error {
	started := time.Now()
	defer func() {
		_ = pp.setStatic("processor_process_blockdata_session_add_operations_elapsed", time.Since(started))
	}()

	if err := pp.blockDataSession.AddOperations(pp.operations...); err != nil {
		return err
	} else if err := pp.blockDataSession.CloseOperations(); err != nil {
		return err
	} else {
		return nil
	}
}

func (pp *DefaultProcessor) processBlockDataSessionSetOperationsTree() error {
	started := time.Now()
	defer func() {
		_ = pp.setStatic("processor_process_set_operations_tree_elapsed", time.Since(started))
	}()

	return pp.blockDataSession.SetOperationsTree(pp.operationsTree)
}

func (pp *DefaultProcessor) processBlockDataSessionSetStatesTree() error {
	started := time.Now()
	defer func() {
		_ = pp.setStatic("processor_process_set_states_tree_elapsed", time.Since(started))
	}()

	return pp.blockDataSession.SetStatesTree(pp.statesTree)
}

func (pp *DefaultProcessor) processBlockDataSessionSetStates() error {
	started := time.Now()
	defer func() {
		_ = pp.setStatic("processor_process_set_states_elapsed", time.Since(started)).
			setStatic("processor_process_set_states_number", len(pp.states))
	}()

	if err := pp.blockDataSession.AddStates(pp.states...); err != nil {
		return err
	} else if err := pp.blockDataSession.CloseStates(); err != nil {
		return err
	} else {
		return nil
	}
}

func (pp *DefaultProcessor) processOperations(ctx context.Context) error {
	started := time.Now()
	defer func() {
		_ = pp.setStatic("processor_process_operations_elapsed", time.Since(started))
	}()

	if len(pp.operations) < 1 {
		pp.Log().Debug().Msg("trying to process operations, but empty")

		return nil
	}

	pool, err := storage.NewStatepool(pp.database)
	if err != nil {
		return err
	}
	defer pool.Done()

	if len(pp.operations) > 0 {
		var err error
		if err = pp.processStatesTree(ctx, pool); err != nil {
			return err
		}
	}

	pp.Log().Debug().Int("operations", len(pp.operations)).Msg("operations processed")

	return nil
}

func (pp *DefaultProcessor) prepareBlock(context.Context) error {
	started := time.Now()
	defer func() {
		_ = pp.setStatic("processor_prepare_block_elapsed", time.Since(started))
	}()

	var opsHash, stsHash valuehash.Hash
	if pp.operationsTree.Len() > 0 {
		opsHash = valuehash.NewBytes(pp.operationsTree.Root())
	}
	if pp.statesTree.Len() > 0 {
		stsHash = valuehash.NewBytes(pp.statesTree.Root())
	}

	var blk block.BlockUpdater
	if b, err := block.NewBlockV0(
		pp.suffrageInfo, pp.Fact().Height(), pp.Fact().Round(), pp.Fact().Hash(), pp.baseManifest.Hash(),
		opsHash, stsHash, pp.Fact().ProposedAt(),
	); err != nil {
		return err
	} else if err := pp.blockDataSession.SetManifest(b.Manifest()); err != nil {
		return err
	} else {
		blk = b
	}

	blk = blk.SetOperationsTree(pp.operationsTree).SetOperations(pp.operations).
		SetStatesTree(pp.statesTree).SetStates(pp.states).
		SetINITVoteproof(pp.initVoteproof).SetProposal(pp.sfs)

	pp.blk = blk

	pp.Log().Debug().Object("block", blk).Msg("block generated")

	return nil
}

func (pp *DefaultProcessor) prepareDatabaseSession(ctx context.Context) error {
	started := time.Now()
	defer func() {
		_ = pp.setStatic("processor_prepare_database_session_elapsed", time.Since(started))
	}()

	bs, err := pp.database.NewSession(pp.blk)
	if err != nil {
		return err
	}

	if err := bs.SetBlock(ctx, pp.blk); err != nil {
		pp.Log().Error().Err(err).Msg("failed to store to DatabaseSession")

		return err
	}

	pp.ss = bs

	pp.Log().Debug().Msg("stored to DatabaseSession")

	return nil
}

func (pp *DefaultProcessor) processStatesTree(ctx context.Context, pool *storage.Statepool) error {
	started := time.Now()
	defer func() {
		_ = pp.setStatic("processor_process_operations_states_tree_elapsed", time.Since(started))
	}()

	pp.operationsTree = tree.EmptyFixedTree()
	pp.statesTree = tree.EmptyFixedTree()
	pp.states = nil

	var co *prprocessor.ConcurrentOperationsProcessor
	size := len(pp.operations)
	c, err := prprocessor.NewConcurrentOperationsProcessor(uint64(size), int64(size), pool, pp.oprHintset)
	if err != nil {
		return err
	}
	_ = c.SetLogging(pp.Logging)

	co = c.Start(
		ctx,
		func(sp state.Processor) error {
			switch found, err := pp.database.HasOperationFact(sp.(operation.Operation).Fact().Hash()); {
			case err != nil:
				return err
			case found:
				return operation.NewBaseReasonError("known operation")
			default:
				return nil
			}
		},
	)

	if err := pp.concurrentProcessStatesTree(co, pool); err != nil {
		pp.Log().Error().Err(err).Msg("failed to process statesTree")

		return err
	}

	pp.Log().Debug().Msg("processed statesTree")

	return nil
}

func (pp *DefaultProcessor) concurrentProcessStatesTree(
	co *prprocessor.ConcurrentOperationsProcessor,
	pool *storage.Statepool,
) error {
	donech := make(chan error, 1)
	go func() {
		for i := range pp.operations {
			op := pp.operations[i]

			pp.Log().Trace().Stringer("fact", op.Fact().Hash()).Msg("process fact")

			if err := co.Process(uint64(i), op); err != nil {
				donech <- err

				return
			}
		}

		donech <- nil
	}()

	if err := <-donech; err != nil {
		return err
	}

	closedch := make(chan error)
	go func() {
		closedch <- co.Close()
	}()

	select {
	case <-pp.prepareCtx.Done():
		return pp.prepareCtx.Err()
	case err := <-closedch:
		if err != nil {
			return err
		}
	}

	if pool.IsUpdated() {
		tr, states, err := co.StatesTree()
		if err != nil {
			return err
		}

		pp.statesTree = tr
		pp.states = states
	}

	tr, err := co.OperationsTree()
	if err != nil {
		return err
	}

	added := pool.AddedOperations()
	for i := range added {
		pp.operations = append(pp.operations, added[i])
	}

	pp.operationsTree = tr

	return nil
}

func (pp *DefaultProcessor) prepareBlockDataSession(context.Context) error {
	started := time.Now()
	defer func() {
		_ = pp.setStatic("processor_prepare_blockdata_session_elapsed", time.Since(started))
	}()

	i, err := pp.blockData.NewSession(pp.Fact().Height())
	if err != nil {
		pp.Log().Error().Err(err).Msg("failed to make new block database session")

		return err
	}
	pp.blockDataSession = i

	if vp := pp.initVoteproof; vp != nil {
		if err := pp.blockDataSession.SetINITVoteproof(vp); err != nil {
			return err
		}
	}

	if err := pp.blockDataSession.SetSuffrageInfo(pp.suffrageInfo); err != nil {
		return err
	}

	if err := pp.blockDataSession.SetProposal(pp.sfs); err != nil {
		return err
	}

	pp.Log().Debug().Msg("block database session prepared")

	return nil
}

func (pp *DefaultProcessor) resetPrepare() error {
	pp.Log().Debug().Stringer("state", pp.state).Msg("prepare will be resetted")

	if pp.blockDataSession != nil {
		if err := pp.blockDataSession.Cancel(); err != nil {
			return err
		}
	}

	pp.ss = nil
	pp.blockDataSession = nil
	pp.blk = nil
	pp.operations = nil
	pp.operationsTree = tree.EmptyFixedTree()
	pp.states = nil
	pp.statesTree = tree.EmptyFixedTree()

	return pp.resetSave()
}

func (pp *DefaultProcessor) cancelPrepare() error {
	pp.Log().Debug().Stringer("state", pp.state).Msg("prepare will be canceled")

	if pp.blockDataSession != nil {
		if err := pp.blockDataSession.Cancel(); err != nil {
			return err
		}
	}

	pp.ss = nil
	pp.blockDataSession = nil
	pp.blk = nil
	pp.operations = nil
	pp.operationsTree = tree.EmptyFixedTree()
	pp.states = nil
	pp.statesTree = tree.EmptyFixedTree()

	return nil
}
