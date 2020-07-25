package isaac

import (
	"fmt"
	"sync"
	"time"

	"github.com/spikeekips/avl"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/base/tree"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

type internalDefaultProposalProcessor struct {
	sync.RWMutex
	*logging.Logging
	localstate                *Localstate
	stopped                   bool
	suffrage                  base.Suffrage
	lastManifest              block.Manifest
	block                     block.BlockUpdater
	proposal                  ballot.Proposal
	proposedOperations        map[string]struct{}
	operations                []operation.Operation
	bs                        storage.BlockStorage
	si                        block.SuffrageInfoV0
	operationProcessorHintSet *hint.Hintmap
	operationProcessors       map[hint.Hint]OperationProcessor
}

func newInternalDefaultProposalProcessor(
	localstate *Localstate,
	suffrage base.Suffrage,
	proposal ballot.Proposal,
	operationProcessorHintSet *hint.Hintmap,
) (*internalDefaultProposalProcessor, error) {
	var lastManifest block.Manifest
	switch m, found, err := localstate.Storage().LastManifest(); {
	case !found:
		return nil, storage.NotFoundError.Errorf("last manifest is empty")
	case err != nil:
		return nil, err
	default:
		lastManifest = m
	}

	proposedOperations := map[string]struct{}{}
	for _, h := range proposal.Operations() {
		proposedOperations[h.String()] = struct{}{}
	}

	var si block.SuffrageInfoV0
	{
		var ns []base.Node
		for _, address := range suffrage.Nodes() {
			if address.Equal(localstate.Node().Address()) {
				ns = append(ns, localstate.Node())
			} else if n, found := localstate.Nodes().Node(address); !found {
				return nil, xerrors.Errorf("suffrage node, %s not found in NodePool(Localstate)", address)
			} else {
				ns = append(ns, n)
			}
		}

		si = block.NewSuffrageInfoV0(proposal.Node(), ns)
	}

	return &internalDefaultProposalProcessor{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "internal-proposal-processor-inside-v0")
		}),
		localstate:                localstate,
		suffrage:                  suffrage,
		proposal:                  proposal,
		lastManifest:              lastManifest,
		proposedOperations:        proposedOperations,
		si:                        si,
		operationProcessorHintSet: operationProcessorHintSet,
		operationProcessors:       map[hint.Hint]OperationProcessor{},
	}, nil
}

func (pp *internalDefaultProposalProcessor) stop() {
	pp.Lock()
	defer pp.Unlock()

	pp.stopped = true
	pp.operationProcessors = nil
}

func (pp *internalDefaultProposalProcessor) isStopped() bool {
	pp.RLock()
	defer pp.RUnlock()

	return pp.stopped
}

func (pp *internalDefaultProposalProcessor) processINIT(initVoteproof base.Voteproof) (block.Block, error) {
	if pp.isStopped() {
		return nil, xerrors.Errorf("already stopped")
	}

	errChan := make(chan error)
	blkChan := make(chan block.Block)
	go func() {
		blk, err := pp.process(initVoteproof)
		if err != nil {
			errChan <- err

			return
		}

		blkChan <- blk
	}()

	// FUTURE if timed out, the next proposal may be able to be passed within
	// timeout. The long-taken operations should be checked and eliminated.

	defer pp.stop()

	var blk block.Block
	select {
	case <-time.After(pp.localstate.Policy().TimeoutProcessProposal()):
		return nil, xerrors.Errorf("timeout to process Proposal")
	case err := <-errChan:
		return nil, err
	case blk = <-blkChan:
	}

	return blk, nil
}

func (pp *internalDefaultProposalProcessor) process(initVoteproof base.Voteproof) (block.Block, error) {
	if pp.block != nil {
		return pp.block, nil
	}

	if ops, err := pp.extractOperations(); err != nil {
		return nil, err
	} else {
		pp.operations = ops
	}

	var operationsTree, statesTree *tree.AVLTree
	var operationsHash, statesHash valuehash.Hash
	if len(pp.operations) > 0 {
		var err error
		if statesTree, statesHash, err = pp.processStates(); err != nil {
			return nil, err
		} else if statesTree != nil {
			if operationsTree, operationsHash, err = pp.processOperations(); err != nil {
				return nil, err
			}
		}
	}

	var blk block.BlockUpdater
	if b, err := block.NewBlockV0(
		pp.si,
		pp.proposal.Height(), pp.proposal.Round(), pp.proposal.Hash(), pp.lastManifest.Hash(),
		operationsHash, statesHash,
	); err != nil {
		return nil, err
	} else {
		blk = b
	}

	pp.Log().Debug().
		Dict("block", logging.Dict().
			Hinted("hash", blk.Hash()).
			Hinted("height", blk.Height()).
			Hinted("round", blk.Round()).
			Hinted("proposal_hash", blk.Proposal()).
			Hinted("previous_block", blk.PreviousBlock()).
			Hinted("operations_hash", blk.OperationsHash()).
			Hinted("states_hash", blk.StatesHash()),
		).Msg("processed block")

	if statesTree != nil {
		if err := pp.updateStates(statesTree, blk); err != nil {
			return nil, err
		}
	}

	blk = blk.SetOperations(operationsTree).SetStates(statesTree).SetINITVoteproof(initVoteproof)
	if bs, err := pp.localstate.Storage().OpenBlockStorage(blk); err != nil {
		return nil, err
	} else {
		pp.bs = bs
		pp.block = blk
	}

	return blk, nil
}

func (pp *internalDefaultProposalProcessor) extractOperations() ([]operation.Operation, error) {
	if len(pp.proposal.Seals()) < 1 {
		return nil, nil
	}

	// TODO only defined operations should be filtered from seal, not all
	// operations of seal.

	founds := map[string]operation.Operation{}
	var notFounds []valuehash.Hash
	for _, h := range pp.proposal.Seals() {
		if pp.isStopped() {
			return nil, xerrors.Errorf("already stopped")
		}

		switch ops, found, err := pp.getOperationsFromStorage(h); {
		case err != nil:
			return nil, err
		case !found:
			notFounds = append(notFounds, h)
			continue
		case len(ops) < 1:
			continue
		default:
			for i := range ops {
				op := ops[i]
				if _, found := founds[op.Hash().String()]; found {
					continue
				}

				founds[op.Hash().String()] = op
			}
		}
	}

	if len(notFounds) > 0 {
		ops, err := pp.getOperationsThruChannel(pp.proposal.Node(), notFounds, founds)
		if err != nil {
			return nil, err
		}

		for i := range ops {
			op := ops[i]
			founds[op.Hash().String()] = op
		}
	}

	var ops []operation.Operation
	for _, h := range pp.proposal.Operations() {
		if pp.isStopped() {
			return nil, xerrors.Errorf("already stopped")
		}

		if op, found := founds[h.String()]; !found {
			return nil, xerrors.Errorf("failed to fetch Operation from Proposal: operation=%s", h)
		} else {
			ops = append(ops, op)
		}
	}

	return pp.filterOperations(ops)
}

func (pp *internalDefaultProposalProcessor) processOperations() (*tree.AVLTree, valuehash.Hash, error) {
	tg := avl.NewTreeGenerator()
	for i := range pp.operations {
		if pp.isStopped() {
			return nil, nil, xerrors.Errorf("already stopped")
		}

		if _, err := tg.Add(operation.NewOperationAVLNodeMutable(pp.operations[i])); err != nil {
			return nil, nil, err
		}
	}

	return pp.validateTree(tg)
}

func (pp *internalDefaultProposalProcessor) processStates() (*tree.AVLTree, valuehash.Hash, error) {
	var pool *Statepool
	if p, err := NewStatepool(pp.localstate.Storage()); err != nil {
		return nil, nil, err
	} else {
		pool = p
	}

	// NOTE for performance, gathers OperationProcessors by each operation
	// before processing them.
	var mopr map[string]OperationProcessor
	if m, err := func() (map[string]OperationProcessor, error) {
		pp.Lock()
		defer pp.Unlock()

		m := map[string]OperationProcessor{}
		for i := range pp.operations {
			op := pp.operations[i]
			if opr, err := pp.operationProcessor(op, pool); err != nil {
				return nil, err
			} else {
				m[op.Hash().String()] = opr
			}
		}

		return m, nil
	}(); err != nil {
		return nil, nil, err
	} else {
		mopr = m
	}

	for i := range pp.operations {
		if pp.isStopped() {
			return nil, nil, xerrors.Errorf("already stopped")
		}

		op := pp.operations[i]
		if err := pp.processOperation(op, mopr[op.Hash().String()]); err != nil {
			return nil, nil, err
		}
	}

	return pp.generateStatesTree(pool)
}

func (pp *internalDefaultProposalProcessor) operationProcessor(
	op operation.Operation,
	pool *Statepool,
) (OperationProcessor, error) {
	if opr, found := pp.operationProcessors[op.Hint()]; found {
		return opr, nil
	}

	var opr OperationProcessor
	if hinter, found := pp.operationProcessorHintSet.Get(op); !found {
		opr = defaultOperationProcessor{}
	} else if p, ok := hinter.(OperationProcessor); !ok {
		return nil, xerrors.Errorf("invalid OperationProcessor found, %T", hinter)
	} else {
		opr = p
	}

	opr = opr.New(pool)
	pp.operationProcessors[op.Hint()] = opr

	return opr, nil
}

func (pp *internalDefaultProposalProcessor) processOperation(op operation.Operation, opr OperationProcessor) error {
	var sp state.StateProcessor
	if p, ok := op.(state.StateProcessor); !ok {
		pp.Log().Error().Str("operation_type", fmt.Sprintf("%T", op)).
			Msg("operation does not support state.StateProcessor")

		return nil
	} else {
		sp = p
	}

	switch err := opr.Process(sp); {
	case err == nil:
		return nil
	case xerrors.Is(err, state.IgnoreOperationProcessingError):
		pp.Log().VerboseFunc(func(e *logging.Event) logging.Emitter {
			return e.Err(err).Interface("operation", op)
		}).Hinted("operation_hash", op.Hash()).Msg("operation ignored")

		return nil
	default:
		pp.Log().Error().Err(err).Interface("operation", op).Msg("failed to process operation")

		return err
	}
}

func (pp *internalDefaultProposalProcessor) generateStatesTree(pool *Statepool) (*tree.AVLTree, valuehash.Hash, error) {
	if !pool.IsUpdated() {
		return nil, nil, nil
	}

	tg := avl.NewTreeGenerator()
	for _, s := range pool.Updates() {
		if pp.isStopped() {
			return nil, nil, xerrors.Errorf("already stopped")
		} else if err := s.SetHash(s.GenerateHash()); err != nil {
			return nil, nil, err
		} else if err := s.IsValid(nil); err != nil {
			return nil, nil, err
		} else if _, err := tg.Add(state.NewStateV0AVLNodeMutable(s.(*state.StateV0))); err != nil {
			return nil, nil, err
		}
	}

	return pp.validateTree(tg)
}

func (pp *internalDefaultProposalProcessor) getOperationsFromStorage(h valuehash.Hash) (
	[]operation.Operation, bool, error,
) {
	var osl operation.Seal
	if sl, found, err := pp.localstate.Storage().Seal(h); !found {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	} else if os, ok := sl.(operation.Seal); !ok {
		return nil, false, xerrors.Errorf("not operation.Seal: %T", sl)
	} else {
		osl = os
	}

	founds := map[string]struct{}{}
	var ops []operation.Operation // nolint
	for i := range osl.Operations() {
		if pp.isStopped() {
			return nil, false, xerrors.Errorf("already stopped")
		}

		op := osl.Operations()[i]
		if _, found := pp.proposedOperations[op.Hash().String()]; !found {
			continue
		} else if _, found := founds[op.Hash().String()]; found {
			continue
		} else {
			founds[op.Hash().String()] = struct{}{}
		}

		ops = append(ops, op)
	}

	return ops, true, nil
}

func (pp *internalDefaultProposalProcessor) getOperationsThruChannel(
	proposer base.Address,
	notFounds []valuehash.Hash,
	founds map[string]operation.Operation,
) ([]operation.Operation, error) {
	if pp.localstate.Node().Address().Equal(proposer) {
		pp.Log().Warn().Msg("proposer is local node, but local node should have seals. Hmmm")
	}

	node, found := pp.localstate.Nodes().Node(proposer)
	if !found {
		return nil, xerrors.Errorf("unknown proposer: %v", proposer)
	}

	received, err := node.Channel().Seals(notFounds)
	if err != nil {
		return nil, err
	}

	if err := pp.localstate.Storage().NewSeals(received); err != nil {
		return nil, err
	}

	var ops []operation.Operation
	for i := range received {
		if pp.isStopped() {
			return nil, xerrors.Errorf("already stopped")
		}

		sl := received[i]
		if os, ok := sl.(operation.Seal); !ok {
			return nil, xerrors.Errorf("not operation.Seal: %T", sl)
		} else {
			for i := range os.Operations() {
				op := os.Operations()[i]
				if _, found := pp.proposedOperations[op.Hash().String()]; !found {
					continue
				} else if _, found := founds[op.Hash().String()]; found {
					continue
				}

				ops = append(ops, op)
			}
		}
	}

	return ops, nil
}

func (pp *internalDefaultProposalProcessor) setACCEPTVoteproof(acceptVoteproof base.Voteproof) error {
	pp.Lock()
	defer pp.Unlock()

	if pp.bs == nil {
		return xerrors.Errorf("not yet processed")
	}

	blk := pp.block.SetACCEPTVoteproof(acceptVoteproof)
	if err := pp.bs.SetBlock(blk); err != nil {
		return err
	}
	pp.block = blk

	return nil
}

func (pp *internalDefaultProposalProcessor) validateTree(tg *avl.TreeGenerator) (*tree.AVLTree, valuehash.Hash, error) {
	var tr *tree.AVLTree
	if t, err := tg.Tree(); err != nil {
		return nil, nil, err
	} else if at, err := tree.NewAVLTree(t); err != nil {
		return nil, nil, err
	} else if err := at.IsValid(); err != nil {
		return nil, nil, err
	} else {
		tr = at
	}

	return tr, tr.RootHash(), nil
}

func (pp *internalDefaultProposalProcessor) updateStates(tr *tree.AVLTree, blk block.Block) error {
	return tr.Traverse(func(node tree.Node) (bool, error) {
		if pp.isStopped() {
			return false, xerrors.Errorf("already stopped")
		}

		var st state.StateUpdater
		if s, ok := node.(*state.StateV0AVLNodeMutable); !ok {
			return false, xerrors.Errorf("not state.StateV0AVLNode: %T", node)
		} else {
			st = s.State().(state.StateUpdater)
		}

		if err := st.SetCurrentBlock(blk.Height(), blk.Hash()); err != nil {
			return false, err
		}

		return true, nil
	})
}

func (pp *internalDefaultProposalProcessor) filterOperation(op operation.Operation) (bool, error) {
	// NOTE Duplicated Operation.Fact().Hash, the latter will be ignored.
	switch found, err := pp.localstate.Storage().HasOperation(op.Hash()); {
	case err != nil:
		// TODO check by op.Fact().Hash() instead of op.Hash()
		return false, err
	case found: // already stored Operation
		return false, nil
	default:
		return true, nil
	}
}

func (pp *internalDefaultProposalProcessor) filterOperations(ops []operation.Operation) ([]operation.Operation, error) {
	var nop []operation.Operation // nolint
	founds := map[string]struct{}{}
	for i := range ops {
		if pp.isStopped() {
			return nil, xerrors.Errorf("already stopped")
		}

		op := ops[i]
		if _, found := founds[op.Hash().String()]; found {
			continue
		} else if ok, err := pp.filterOperation(op); err != nil {
			return nil, err
		} else if !ok {
			continue
		}

		nop = append(nop, op)
		founds[op.Hash().String()] = struct{}{}
	}

	return nop, nil
}
