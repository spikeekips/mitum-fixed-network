package isaac

import (
	"context"
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
	localstate    *Localstate
	stopped       bool
	suffrage      base.Suffrage
	lastManifest  block.Manifest
	block         block.BlockUpdater
	proposal      ballot.Proposal
	proposedFacts map[string]struct{}
	operations    []operation.Operation
	bs            storage.BlockStorage
	si            block.SuffrageInfoV0
	oprHintset    *hint.Hintmap
	oprs          map[hint.Hint]OperationProcessor
	statesLock    sync.RWMutex
	stateValues   map[string]interface{}
}

func newInternalDefaultProposalProcessor(
	localstate *Localstate,
	suffrage base.Suffrage,
	proposal ballot.Proposal,
	oprHintset *hint.Hintmap,
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
	for _, h := range proposal.Facts() {
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
		localstate:    localstate,
		suffrage:      suffrage,
		proposal:      proposal,
		lastManifest:  lastManifest,
		proposedFacts: proposedOperations,
		si:            si,
		oprHintset:    oprHintset,
		oprs:          map[hint.Hint]OperationProcessor{},
		stateValues:   map[string]interface{}{},
	}, nil
}

func (pp *internalDefaultProposalProcessor) stop() {
	pp.Lock()
	defer pp.Unlock()

	pp.stopped = true
	pp.oprs = nil
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

	if pp.block != nil {
		return pp.block, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), pp.localstate.Policy().TimeoutProcessProposal())
	defer cancel()

	errChan := make(chan error)
	blkChan := make(chan block.Block)
	go func() {
		s := time.Now()

		blk, err := pp.process(ctx, initVoteproof)
		pp.setState("process", time.Since(s))

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
	case <-ctx.Done():
		return nil, xerrors.Errorf("timeout to process Proposal: %w", ctx.Err())
	case err := <-errChan:
		return nil, err
	case blk = <-blkChan:
	}

	return blk, nil
}

func (pp *internalDefaultProposalProcessor) process(
	ctx context.Context, initVoteproof base.Voteproof,
) (block.Block, error) {
	if len(pp.proposal.Seals()) > 0 {
		if ops, err := pp.extractOperations(); err != nil {
			return nil, err
		} else {
			pp.operations = ops
		}
	}

	var operationsTree, statesTree *tree.AVLTree
	var operationsHash, statesHash valuehash.Hash
	if len(pp.operations) > 0 {
		var err error
		if statesTree, statesHash, err = pp.processStates(ctx); err != nil {
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

	pp.Log().Debug().
		Dict("block", logging.Dict().
			Hinted("hash", blk.Hash()).
			Hinted("height", blk.Height()).
			Hinted("round", blk.Round()).
			Hinted("proposal", blk.Proposal()).
			Hinted("previous_block", blk.PreviousBlock()).
			Hinted("operations_hash", blk.OperationsHash()).
			Hinted("states_hash", blk.StatesHash()),
		).Msg("processed block")

	return blk, nil
}

func (pp *internalDefaultProposalProcessor) extractOperations() ([]operation.Operation, error) {
	s := time.Now()
	defer func() {
		pp.setState("extract-operations", time.Since(s))
	}()

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
				fh := op.Fact().Hash().String()
				if _, found := founds[fh]; found {
					continue
				}

				founds[fh] = op
			}
		}
	}

	if len(notFounds) > 0 {
		if ops, err := pp.getOperationsThruChannel(pp.proposal.Node(), notFounds, founds); err != nil {
			return nil, err
		} else {
			for i := range ops {
				op := ops[i]
				founds[op.Fact().Hash().String()] = op
			}
		}
	}

	var ops []operation.Operation
	for _, h := range pp.proposal.Facts() {
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
	s := time.Now()
	defer func() {
		pp.setState("process-operations", time.Since(s))
	}()

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

func (pp *internalDefaultProposalProcessor) processStates(ctx context.Context) (*tree.AVLTree, valuehash.Hash, error) {
	s := time.Now()
	defer func() {
		pp.setState("process-states", time.Since(s))
	}()

	var pool *Statepool
	if p, err := NewStatepool(pp.localstate.Storage()); err != nil {
		return nil, nil, err
	} else {
		pool = p
	}

	var co *ConcurrentOperationsProcessor
	if c, err := NewConcurrentOperationsProcessor(len(pp.operations), pool, pp.oprHintset); err != nil {
		return nil, nil, err
	} else {
		nctx, cancel := context.WithCancel(ctx)
		defer cancel()

		co = c.Start(nctx)

		go func() {
			<-nctx.Done()
			_ = co.Cancel()
		}()
	}

	for i := range pp.operations {
		if pp.isStopped() {
			return nil, nil, xerrors.Errorf("already stopped")
		} else if err := co.Process(pp.operations[i]); err != nil {
			return nil, nil, err
		}
	}

	if err := co.Close(); err != nil {
		return nil, nil, err
	}

	return pp.generateStatesTree(pool)
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
		} else if _, err := tg.Add(state.NewStateV0AVLNodeMutable(s.(*state.StateV0Updater))); err != nil {
			return nil, nil, err
		}
	}

	return pp.validateTree(tg)
}

func (pp *internalDefaultProposalProcessor) getOperationsFromStorage(h valuehash.Hash) (
	[]operation.Operation, bool, error,
) {
	var osl operation.Seal
	if sl, found, err := pp.localstate.Storage().Seal(h); err != nil {
		return nil, false, err
	} else if !found {
		return nil, false, nil
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
		fh := op.Fact().Hash().String()
		if _, found := pp.proposedFacts[fh]; !found {
			continue
		} else if _, found := founds[fh]; found {
			continue
		} else {
			founds[fh] = struct{}{}
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

		return nil, nil
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
		if !xerrors.Is(err, storage.DuplicatedError) {
			return nil, err
		}
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
				fh := op.Fact().Hash().String()
				if _, found := pp.proposedFacts[fh]; !found {
					continue
				} else if _, found := founds[fh]; found {
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

	s := time.Now()
	defer func() {
		pp.setState("set-accept-voteproof", time.Since(s))
	}()

	var fact ballot.ACCEPTBallotFact
	if m := acceptVoteproof.Majority(); m == nil {
		return xerrors.Errorf("acceptVoteproof has empty majority")
	} else if f, ok := m.(ballot.ACCEPTBallotFact); !ok {
		return xerrors.Errorf("acceptVoteproof does not have ballot.ACCEPTBallotFact")
	} else {
		fact = f
	}

	if !pp.block.Hash().Equal(fact.NewBlock()) {
		return xerrors.Errorf("hash of the processed block does not match with acceptVoteproof")
	}

	blk := pp.block.SetACCEPTVoteproof(acceptVoteproof).
		SetProposal(pp.proposal)

	if err := func() error {
		s := time.Now()
		defer func() {
			pp.setState("set-block", time.Since(s))
		}()

		return pp.bs.SetBlock(blk)
	}(); err != nil {
		return err
	}

	if seals := pp.proposal.Seals(); len(seals) > 0 {
		// TODO when failed, seals of UnstageOperationSeals should be recovered
		if err := func() error {
			s := time.Now()
			defer func() {
				pp.setState("unstage-operation-seals", time.Since(s))
			}()

			return pp.bs.UnstageOperationSeals(seals)
		}(); err != nil {
			return err
		}
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
	s := time.Now()
	defer func() {
		pp.setState("update-states", time.Since(s))
	}()

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
	switch found, err := pp.localstate.Storage().HasOperationFact(op.Fact().Hash()); {
	case err != nil:
		return false, err
	case found: // already stored Operation
		return false, nil
	default:
		return true, nil
	}
}

func (pp *internalDefaultProposalProcessor) filterOperations(ops []operation.Operation) ([]operation.Operation, error) {
	var nop []operation.Operation // nolint
	for i := range ops {
		if pp.isStopped() {
			return nil, xerrors.Errorf("already stopped")
		}

		op := ops[i]
		if ok, err := pp.filterOperation(op); err != nil {
			return nil, err
		} else if !ok {
			continue
		}

		nop = append(nop, op)
	}

	return nop, nil
}

func (pp *internalDefaultProposalProcessor) setState(k string, v interface{}) {
	pp.statesLock.Lock()
	defer pp.statesLock.Unlock()

	pp.stateValues[k] = v
}

func (pp *internalDefaultProposalProcessor) states() map[string]interface{} {
	pp.statesLock.RLock()
	defer pp.statesLock.RUnlock()

	return pp.stateValues
}
