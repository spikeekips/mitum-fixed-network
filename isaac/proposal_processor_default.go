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
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

type DefaultProposalProcessor struct {
	*logging.Logging
	localstate *Localstate
	processors *sync.Map
	suffrage   base.Suffrage
}

func NewDefaultProposalProcessor(localstate *Localstate, suffrage base.Suffrage) *DefaultProposalProcessor {
	return &DefaultProposalProcessor{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "default-proposal-processor")
		}),
		localstate: localstate,
		suffrage:   suffrage,
		processors: &sync.Map{},
	}
}

func (dp *DefaultProposalProcessor) Initialize() error {
	return nil
}

func (dp *DefaultProposalProcessor) IsProcessed(ph valuehash.Hash) bool {
	_, found := dp.processors.Load(ph.String())

	return found
}

func (dp *DefaultProposalProcessor) ProcessINIT(ph valuehash.Hash, initVoteproof base.Voteproof) (block.Block, error) {
	if i, found := dp.processors.Load(ph.String()); found {
		processor := i.(*internalDefaultProposalProcessor)

		return processor.block, nil
	}

	if initVoteproof.Stage() != base.StageINIT {
		return nil, xerrors.Errorf("ProcessINIT needs INIT Voteproof")
	}

	var proposal ballot.Proposal
	if pr, err := dp.checkProposal(ph, initVoteproof); err != nil {
		return nil, err
	} else {
		proposal = pr
	}

	processor, err := newInternalDefaultProposalProcessor(dp.localstate, dp.suffrage, proposal)
	if err != nil {
		return nil, err
	}

	_ = processor.SetLogger(dp.Log())

	blk, err := processor.processINIT(initVoteproof)
	if err != nil {
		return nil, err
	}

	dp.processors.Store(ph.String(), processor)

	return blk, nil
}

func (dp *DefaultProposalProcessor) ProcessACCEPT(
	ph valuehash.Hash, acceptVoteproof base.Voteproof,
) (storage.BlockStorage, error) {
	if acceptVoteproof.Stage() != base.StageACCEPT {
		return nil, xerrors.Errorf("Processaccept needs ACCEPT Voteproof")
	}

	var processor *internalDefaultProposalProcessor
	if i, found := dp.processors.Load(ph.String()); !found {
		return nil, xerrors.Errorf("not processed ProcessINIT")
	} else {
		processor = i.(*internalDefaultProposalProcessor)
	}

	if err := processor.setACCEPTVoteproof(acceptVoteproof); err != nil {
		return nil, err
	}

	defer dp.processors.Delete(ph.String())

	return processor.bs, nil
}

func (dp *DefaultProposalProcessor) checkProposal(
	ph valuehash.Hash, initVoteproof base.Voteproof,
) (ballot.Proposal, error) {
	var proposal ballot.Proposal
	if sl, found, err := dp.localstate.Storage().Seal(ph); !found {
		return nil, storage.NotFoundError.Errorf("seal not found")
	} else if err != nil {
		return nil, err
	} else if pr, ok := sl.(ballot.Proposal); !ok {
		return nil, xerrors.Errorf("seal is not Proposal: %T", sl)
	} else {
		proposal = pr
	}

	timespan := dp.localstate.Policy().TimespanValidBallot()
	if proposal.SignedAt().Before(initVoteproof.FinishedAt().Add(timespan * -1)) {
		return nil, xerrors.Errorf(
			"Proposal was sent before Voteproof; SignedAt=%s now=%s timespan=%s",
			proposal.SignedAt(), initVoteproof.FinishedAt(), timespan,
		)
	}

	return proposal, nil
}

type internalDefaultProposalProcessor struct {
	*logging.Logging
	localstate         *Localstate
	suffrage           base.Suffrage
	lastManifest       block.Manifest
	block              block.BlockUpdater
	proposal           ballot.Proposal
	proposedOperations map[string]struct{}
	operations         []state.OperationInfoV0
	bs                 storage.BlockStorage
	si                 block.SuffrageInfoV0
}

func newInternalDefaultProposalProcessor(
	localstate *Localstate,
	suffrage base.Suffrage,
	proposal ballot.Proposal,
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
		localstate:         localstate,
		suffrage:           suffrage,
		proposal:           proposal,
		lastManifest:       lastManifest,
		proposedOperations: proposedOperations,
		si:                 si,
	}, nil
}

func (pp *internalDefaultProposalProcessor) processINIT(initVoteproof base.Voteproof) (block.Block, error) {
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

	var operationsTree, statesTree *tree.AVLTree
	var operationsHash, statesHash valuehash.Hash
	if tr, err := pp.processOperations(); err != nil {
		return nil, err
	} else if tr != nil {
		if h, err := tr.RootHash(); err != nil {
			return nil, err
		} else {
			operationsTree, operationsHash = tr, h
		}
	}

	if tr, err := pp.processStates(); err != nil {
		return nil, err
	} else if tr != nil {
		if h, err := tr.RootHash(); err != nil {
			return nil, err
		} else {
			statesTree, statesHash = tr, h
		}
	}

	var blk block.BlockUpdater
	if b, err := block.NewBlockV0(
		pp.si,
		pp.proposal.Height(), pp.proposal.Round(), pp.proposal.Hash(), pp.lastManifest.Hash(),
		operationsHash,
		statesHash,
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

	blk = blk.
		SetOperations(operationsTree).
		SetStates(statesTree).
		SetINITVoteproof(initVoteproof)

	if bs, err := pp.localstate.Storage().OpenBlockStorage(blk); err != nil {
		return nil, err
	} else {
		pp.bs = bs
	}

	pp.block = blk

	return blk, nil
}

func (pp *internalDefaultProposalProcessor) extractOperations() ([]state.OperationInfoV0, error) {
	// NOTE the order of operation should be kept by the order of
	// Proposal.Seals()
	founds := map[string]state.OperationInfoV0{}

	var notFounds []valuehash.Hash
	for _, h := range pp.proposal.Seals() {
		ops, err := pp.getOperationsFromStorage(h)
		if err != nil {
			if storage.IsNotFoundError(err) {
				notFounds = append(notFounds, h)
				continue
			}

			return nil, err
		} else if len(ops) < 1 {
			continue
		}

		for i := range ops {
			if ops[i].Operation() == nil {
				continue
			}

			op := ops[i]
			founds[op.Operation().String()] = op
		}
	}

	if len(notFounds) > 0 {
		ops, err := pp.getOperationsThruChannel(pp.proposal.Node(), notFounds, founds)
		if err != nil {
			return nil, err
		}

		for i := range ops {
			op := ops[i]
			founds[op.Operation().String()] = op
		}
	}

	var operations []state.OperationInfoV0
	for _, h := range pp.proposal.Operations() {
		if oi, found := founds[h.String()]; !found {
			return nil, xerrors.Errorf("failed to fetch Operation from Proposal: operation=%s", h)
		} else {
			operations = append(operations, oi)
		}
	}

	return operations, nil
}

func (pp *internalDefaultProposalProcessor) processOperations() (*tree.AVLTree, error) {
	if len(pp.proposal.Seals()) < 1 {
		return nil, nil
	}

	var operations []state.OperationInfoV0

	if ops, err := pp.extractOperations(); err != nil {
		return nil, err
	} else {
		founds := map[string]struct{}{}

		for i := range ops {
			op := ops[i]
			// NOTE Duplicated Operation.Hash, the latter will be ignored.
			if _, found := founds[op.Operation().String()]; found {
				continue
			} else if found, err := pp.localstate.Storage().HasOperation(op.Operation()); err != nil {
				return nil, err
			} else if found { // already stored Operation
				continue
			}

			operations = append(operations, op)
			founds[op.Operation().String()] = struct{}{}
		}
	}

	if len(operations) < 1 {
		return nil, nil
	}

	tg := avl.NewTreeGenerator()
	for i := range operations {
		op := operations[i]
		n := operation.NewOperationAVLNodeMutable(op.RawOperation())
		if _, err := tg.Add(n); err != nil {
			return nil, err
		}
	}

	tr, err := pp.validateTree(tg)
	if err != nil {
		return nil, err
	}

	pp.operations = operations

	return tr, nil
}

func (pp *internalDefaultProposalProcessor) processStates() (*tree.AVLTree, error) {
	var pool *StatePool
	if p, err := NewStatePool(pp.localstate.Storage()); err != nil {
		return nil, err
	} else {
		pool = p
	}

	for i := range pp.operations {
		opi := pp.operations[i]
		op := opi.RawOperation()
		opp, ok := op.(state.OperationProcesser)
		if !ok {
			pp.Log().Error().
				Str("operation_type", fmt.Sprintf("%T", op)).
				Msg("operation does not support state.OperationProcesser")
			continue
		}

		if err := opp.ProcessOperation(
			pool.Get,
			func(s state.StateUpdater) error {
				if err := s.AddOperationInfo(opi); err != nil {
					return err
				}

				return pool.Set(s)
			},
		); err != nil {
			pp.Log().Error().Err(err).
				Interface("operation", op).
				Msg("failed to process operation")

			continue
		}
	}

	updated := pool.Updated()
	if len(updated) < 1 {
		return nil, nil
	}

	tg := avl.NewTreeGenerator()
	for _, s := range updated {
		if err := s.SetHash(s.GenerateHash()); err != nil {
			return nil, err
		}

		if err := s.IsValid(nil); err != nil {
			return nil, err
		}

		n := state.NewStateV0AVLNodeMutable(s.(*state.StateV0))
		if _, err := tg.Add(n); err != nil {
			return nil, err
		}
	}

	return pp.validateTree(tg)
}

func (pp *internalDefaultProposalProcessor) getOperationsFromStorage(h valuehash.Hash) (
	[]state.OperationInfoV0, error,
) {
	var osl operation.Seal
	if sl, found, err := pp.localstate.Storage().Seal(h); !found {
		return nil, storage.NotFoundError.Errorf("seal not found")
	} else if err != nil {
		return nil, err
	} else if os, ok := sl.(operation.Seal); !ok {
		return nil, xerrors.Errorf("not operation.Seal: %T", sl)
	} else {
		osl = os
	}

	ops := make([]state.OperationInfoV0, len(osl.Operations()))
	for i, op := range osl.Operations() {
		if _, found := pp.proposedOperations[op.Hash().String()]; !found {
			continue
		}

		ops[i] = state.NewOperationInfoV0(op, h)
	}

	return ops, nil
}

func (pp *internalDefaultProposalProcessor) getOperationsThruChannel(
	proposer base.Address,
	notFounds []valuehash.Hash,
	founds map[string]state.OperationInfoV0,
) ([]state.OperationInfoV0, error) {
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

	var ops []state.OperationInfoV0
	for _, sl := range received {
		if os, ok := sl.(operation.Seal); !ok {
			return nil, xerrors.Errorf("not operation.Seal: %T", sl)
		} else {
			for _, op := range os.Operations() {
				if _, found := pp.proposedOperations[op.Hash().String()]; !found {
					continue
				} else if _, found := founds[op.Hash().String()]; found {
					continue
				}

				ops = append(ops, state.NewOperationInfoV0(op, sl.Hash()))
			}
		}
	}

	return ops, nil
}

func (pp *internalDefaultProposalProcessor) setACCEPTVoteproof(acceptVoteproof base.Voteproof) error {
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

func (pp *internalDefaultProposalProcessor) validateTree(tg *avl.TreeGenerator) (*tree.AVLTree, error) {
	var tr *tree.AVLTree
	if t, err := tg.Tree(); err != nil {
		return nil, err
	} else if at, err := tree.NewAVLTree(t); err != nil {
		return nil, err
	} else if err := at.IsValid(); err != nil {
		return nil, err
	} else {
		tr = at
	}

	return tr, nil
}

func (pp *internalDefaultProposalProcessor) updateStates(tr *tree.AVLTree, blk block.Block) error {
	return tr.Traverse(func(node tree.Node) (bool, error) {
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
