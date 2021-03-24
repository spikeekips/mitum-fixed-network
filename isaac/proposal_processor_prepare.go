package isaac

import (
	"bytes"
	"context"
	"sort"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/tree"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

var blockDataMapContextKey util.ContextKey = "blockdata_map"

func (pp *DefaultProcessor) Prepare(ctx context.Context) (block.Block, error) {
	pp.Lock()
	defer pp.Unlock()

	if err := pp.resetPrepare(); err != nil {
		return nil, err
	}

	pp.setState(prprocessor.Preparing)

	if err := pp.prepare(ctx); err != nil {
		pp.setState(prprocessor.PrepareFailed)

		if err0 := pp.resetPrepare(); err0 != nil {
			return nil, err0
		}

		return nil, err
	} else {
		pp.setState(prprocessor.Prepared)

		return pp.blk, nil
	}
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
		pp.prepareStorageSession,
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
	ev := pp.Log().Debug().Int("seals", len(pp.proposal.Seals()))

	seals := pp.proposal.Seals()
	if len(seals) < 1 {
		return nil
	}

	se := NewSealsExtracter(pp.nodepool.Local().Address(), pp.proposal.Node(), pp.st, pp.nodepool, seals)
	_ = se.SetLogger(pp.Log())

	if ops, err := se.Extract(ctx); err != nil {
		return xerrors.Errorf("failed to extract seals: %w", err)
	} else {
		ev.Int("operations", len(ops)).Msg("operations extracted from seals of proposal")

		pp.operations = ops

		return nil
	}
}

func (pp *DefaultProcessor) process(ctx context.Context) error {
	if err := pp.blockDataSession.AddOperations(pp.operations...); err != nil {
		return err
	} else if err := pp.blockDataSession.CloseOperations(); err != nil {
		return err
	}

	if err := pp.processOperations(ctx); err != nil {
		return err
	}

	if err := pp.blockDataSession.SetOperationsTree(pp.operationsTree); err != nil {
		return err
	} else if err := pp.blockDataSession.SetStatesTree(pp.statesTree); err != nil {
		return err
	}

	if err := pp.blockDataSession.AddStates(pp.states...); err != nil {
		return err
	} else if err := pp.blockDataSession.CloseStates(); err != nil {
		return err
	}

	return nil
}

func (pp *DefaultProcessor) processOperations(ctx context.Context) error {
	if len(pp.operations) < 1 {
		pp.Log().Debug().Msg("trying to process operations, but empty")

		return nil
	}

	var pool *storage.Statepool
	if p, err := storage.NewStatepool(pp.st); err != nil {
		return err
	} else {
		pool = p
	}

	if len(pp.operations) > 0 {
		var err error
		if err = pp.processStatesTree(ctx, pool); err != nil {
			return err
		}
	}

	if err := pp.processOperationsTree(ctx, pool); err != nil {
		return err
	}

	pp.Log().Debug().Int("operations", len(pp.operations)).Msg("operations processed")

	return nil
}

func (pp *DefaultProcessor) prepareBlock(context.Context) error {
	var opsHash, stsHash valuehash.Hash
	if !pp.operationsTree.IsEmpty() {
		opsHash = valuehash.NewBytes(pp.operationsTree.Root())
	}
	if !pp.statesTree.IsEmpty() {
		stsHash = valuehash.NewBytes(pp.statesTree.Root())
	}

	var blk block.BlockUpdater
	if b, err := block.NewBlockV0(
		pp.suffrageInfo, pp.proposal.Height(), pp.proposal.Round(), pp.proposal.Hash(), pp.baseManifest.Hash(),
		opsHash, stsHash, pp.proposal.SignedAt(),
	); err != nil {
		return err
	} else if err := pp.blockDataSession.SetManifest(b.Manifest()); err != nil {
		return err
	} else {
		blk = b
	}

	blk = blk.SetOperationsTree(pp.operationsTree).SetOperations(pp.operations).
		SetStatesTree(pp.statesTree).SetStates(pp.states).
		SetINITVoteproof(pp.initVoteproof).SetProposal(pp.proposal)

	pp.blk = blk

	pp.Log().Debug().
		Dict("block", logging.Dict().
			Hinted("hash", blk.Hash()).Hinted("height", blk.Height()).Hinted("round", blk.Round()).
			Hinted("proposal", blk.Proposal()).Hinted("previous_block", blk.PreviousBlock()).
			Hinted("operations_hash", blk.OperationsHash()).Hinted("states_hash", blk.StatesHash()),
		).Msg("block generated")

	return nil
}

func (pp *DefaultProcessor) prepareStorageSession(ctx context.Context) error {
	var bs storage.StorageSession
	if b, err := pp.st.NewStorageSession(pp.blk); err != nil {
		return err
	} else {
		bs = b
	}

	if err := bs.SetBlock(ctx, pp.blk); err != nil {
		pp.Log().Error().Err(err).Msg("failed to store to StorageSession")

		return err
	}

	pp.ss = bs

	pp.Log().Debug().Msg("stored to StorageSession")

	return nil
}

func (pp *DefaultProcessor) processStatesTree(ctx context.Context, pool *storage.Statepool) error {
	pp.statesTree = tree.FixedTree{}
	pp.states = nil

	var co *prprocessor.ConcurrentOperationsProcessor
	if c, err := prprocessor.NewConcurrentOperationsProcessor(len(pp.operations), pool, pp.oprHintset); err != nil {
		return err
	} else {
		_ = c.SetLogger(pp.Log())

		co = c.Start(
			ctx,
			func(sp state.Processor) error {
				switch found, err := pp.st.HasOperationFact(sp.(operation.Operation).Fact().Hash()); {
				case err != nil:
					return err
				case found:
					return util.IgnoreError.Errorf("already known")
				default:
					return nil
				}
			},
		)
	}

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
	for i := range pp.operations {
		op := pp.operations[i]

		pp.Log().Verbose().Hinted("fact", op.Fact().Hash()).Msg("process fact")

		if err := co.Process(op); err != nil {
			return err
		}
	}

	if err := co.Close(); err != nil {
		return err
	}

	if !pool.IsUpdated() {
		return nil
	}

	if tr, states, err := pp.generateStatesTree(pool); err != nil {
		return err
	} else {
		pp.statesTree = tr
		pp.states = states

		return nil
	}
}

func (pp *DefaultProcessor) processOperationsTree(_ context.Context, pool *storage.Statepool) error {
	added := pool.AddedOperations()
	for k := range added {
		pp.operations = append(pp.operations, added[k])
	}

	if len(pp.operations) < 1 {
		return nil
	}

	statesOps := pool.InsertedOperations()
	pp.operationsTree = tree.FixedTree{}

	tg := tree.NewFixedTreeGenerator(uint(len(pp.operations)), nil)
	for i := range pp.operations {
		fh := pp.operations[i].Fact().Hash()

		var mod []byte
		if _, found := statesOps[fh.String()]; found {
			mod = base.FactMode2bytes(base.FInStates)
		}

		if err := tg.Add(i, fh.Bytes(), mod); err != nil {
			return err
		}
	}

	if tr, err := tg.Tree(); err != nil {
		pp.Log().Error().Err(err).Msg("failed to process operationsTree")

		return err
	} else {
		pp.operationsTree = tr

		pp.Log().Debug().Msg("processed operationsTree")

		return nil
	}
}

func (pp *DefaultProcessor) generateStatesTree(
	pool *storage.Statepool,
) (tree.FixedTree, []state.State, error) {
	states := make([]state.State, len(pool.Updates()))
	for i, s := range pool.Updates() {
		st := s.GetState()
		if ust, err := st.SetHash(st.GenerateHash()); err != nil {
			return tree.FixedTree{}, nil, err
		} else if err := ust.IsValid(nil); err != nil {
			return tree.FixedTree{}, nil, err
		} else {
			states[i] = ust
		}
	}

	sort.Slice(states, func(i, j int) bool {
		return bytes.Compare(states[i].Hash().Bytes(), states[j].Hash().Bytes()) < 0
	})

	tg := tree.NewFixedTreeGenerator(uint(len(pool.Updates())), nil)
	for i := range states {
		if err := tg.Add(i, states[i].Hash().Bytes(), nil); err != nil {
			return tree.FixedTree{}, nil, err
		}
	}

	if tr, err := tg.Tree(); err != nil {
		return tree.FixedTree{}, nil, err
	} else {
		return tr, states, nil
	}
}

func (pp *DefaultProcessor) prepareBlockDataSession(context.Context) error {
	if i, err := pp.blockData.NewSession(pp.proposal.Height()); err != nil {
		pp.Log().Error().Err(err).Msg("failed to make new block storage session")

		return err
	} else {
		pp.blockDataSession = i
	}

	if vp := pp.initVoteproof; vp != nil {
		if err := pp.blockDataSession.SetINITVoteproof(vp); err != nil {
			return err
		}
	}

	if err := pp.blockDataSession.SetSuffrageInfo(pp.suffrageInfo); err != nil {
		return err
	}

	if err := pp.blockDataSession.SetProposal(pp.proposal); err != nil {
		return err
	}

	pp.Log().Debug().Msg("block storage session prepared")

	return nil
}

func (pp *DefaultProcessor) resetPrepare() error {
	pp.Log().Debug().Str("state", pp.state.String()).Msg("prepare will be resetted")

	if pp.blockDataSession != nil {
		if err := pp.blockDataSession.Cancel(); err != nil {
			return err
		}
	}

	pp.ss = nil
	pp.blockDataSession = nil
	pp.blk = nil
	pp.operations = nil
	pp.operationsTree = tree.FixedTree{}
	pp.states = nil
	pp.statesTree = tree.FixedTree{}

	return pp.resetSave()
}
