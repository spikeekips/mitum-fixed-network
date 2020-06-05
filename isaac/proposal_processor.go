package isaac

import (
	"fmt"
	"sync"

	"github.com/spikeekips/avl"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/base/tree"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/logging"
)

type ProposalProcessor interface {
	IsProcessed(valuehash.Hash /* proposal.Hash() */) bool
	ProcessINIT(valuehash.Hash /* Proposal.Hash() */, base.Voteproof /* INIT Voteproof */) (block.Block, error)
	ProcessACCEPT(
		valuehash.Hash /* Proposal.Hash() */, base.Voteproof, /* ACCEPT Voteproof */
	) (storage.BlockStorage, error)
}

type ProposalProcessorV0 struct {
	*logging.Logging
	localstate *Localstate
	processors *sync.Map
	suffrage   base.Suffrage
}

func NewProposalProcessorV0(localstate *Localstate, suffrage base.Suffrage) *ProposalProcessorV0 {
	return &ProposalProcessorV0{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "proposal-processor-v0")
		}),
		localstate: localstate,
		suffrage:   suffrage,
		processors: &sync.Map{},
	}
}

func (dp *ProposalProcessorV0) IsProcessed(ph valuehash.Hash) bool {
	_, found := dp.processors.Load(ph)

	return found
}

func (dp *ProposalProcessorV0) ProcessINIT(ph valuehash.Hash, initVoteproof base.Voteproof) (block.Block, error) {
	if i, found := dp.processors.Load(ph); found {
		processor := i.(*proposalProcessorV0)

		return processor.block, nil
	}

	if initVoteproof.Stage() != base.StageINIT {
		return nil, xerrors.Errorf("ProcessINIT needs INIT Voteproof")
	}

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

	{
		timespan := dp.localstate.Policy().TimespanValidBallot()
		if proposal.SignedAt().Before(initVoteproof.FinishedAt().Add(timespan * -1)) {
			return nil, xerrors.Errorf(
				"Proposal was sent before Voteproof; SignedAt=%s now=%s timespan=%s",
				proposal.SignedAt(), initVoteproof.FinishedAt(), timespan,
			)
		}
	}

	processor, err := newProposalProcessorV0(dp.localstate, dp.suffrage, proposal)
	if err != nil {
		return nil, err
	}

	_ = processor.SetLogger(dp.Log())

	blk, err := processor.processINIT(initVoteproof)
	if err != nil {
		return nil, err
	}

	dp.processors.Store(ph, processor)

	return blk, nil
}

func (dp *ProposalProcessorV0) ProcessACCEPT(
	ph valuehash.Hash, acceptVoteproof base.Voteproof,
) (storage.BlockStorage, error) {
	if acceptVoteproof.Stage() != base.StageACCEPT {
		return nil, xerrors.Errorf("Processaccept needs ACCEPT Voteproof")
	}

	var processor *proposalProcessorV0
	if i, found := dp.processors.Load(ph); !found {
		return nil, xerrors.Errorf("not processed ProcessINIT")
	} else {
		processor = i.(*proposalProcessorV0)
	}

	if err := processor.setACCEPTVoteproof(acceptVoteproof); err != nil {
		return nil, err
	}

	defer dp.processors.Delete(ph)

	return processor.bs, nil
}

type proposalProcessorV0 struct {
	*logging.Logging
	localstate         *Localstate
	suffrage           base.Suffrage
	lastManifest       block.Manifest
	block              block.BlockUpdater
	proposal           ballot.Proposal
	proposedOperations map[valuehash.Hash]struct{}
	operations         []state.OperationInfoV0
	bs                 storage.BlockStorage
	si                 block.SuffrageInfoV0
}

func newProposalProcessorV0(
	localstate *Localstate,
	suffrage base.Suffrage,
	proposal ballot.Proposal,
) (*proposalProcessorV0, error) {
	var lastManifest block.Manifest
	switch m, found, err := localstate.Storage().LastManifest(); {
	case !found:
		return nil, storage.NotFoundError.Errorf("last manifest is empty")
	case err != nil:
		return nil, err
	default:
		lastManifest = m
	}

	proposedOperations := map[valuehash.Hash]struct{}{}
	for _, h := range proposal.Operations() {
		proposedOperations[h] = struct{}{}
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

	return &proposalProcessorV0{
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

func (pp *proposalProcessorV0) processINIT(initVoteproof base.Voteproof) (block.Block, error) {
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
			operationsTree = tr
			operationsHash = h
		}
	}

	if tr, err := pp.processStates(); err != nil {
		return nil, err
	} else if tr != nil {
		if h, err := tr.RootHash(); err != nil {
			return nil, err
		} else {
			statesTree = tr
			statesHash = h
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

func (pp *proposalProcessorV0) extractOperations() ([]state.OperationInfoV0, error) {
	// NOTE the order of operation should be kept by the order of
	// Proposal.Seals()
	founds := map[valuehash.Hash]state.OperationInfoV0{}

	var notFounds []valuehash.Hash
	for _, h := range pp.proposal.Seals() {
		ops, err := pp.getOperationsFromStorage(h)
		if err != nil {
			if xerrors.Is(err, storage.NotFoundError) {
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
			founds[op.Operation()] = op
		}
	}

	if len(notFounds) > 0 {
		ops, err := pp.getOperationsThruChannel(pp.proposal.Node(), notFounds, founds)
		if err != nil {
			return nil, err
		}

		for i := range ops {
			op := ops[i]
			founds[op.Operation()] = op
		}
	}

	var operations []state.OperationInfoV0
	for _, h := range pp.proposal.Operations() {
		if oi, found := founds[h]; !found {
			return nil, xerrors.Errorf("failed to fetch Operation from Proposal: operation=%s", h)
		} else {
			operations = append(operations, oi)
		}
	}

	return operations, nil
}

func (pp *proposalProcessorV0) processOperations() (*tree.AVLTree, error) {
	if len(pp.proposal.Seals()) < 1 {
		return nil, nil
	}

	var operations []state.OperationInfoV0

	if ops, err := pp.extractOperations(); err != nil {
		return nil, err
	} else {
		founds := map[valuehash.Hash]struct{}{}

		for i := range ops {
			op := ops[i]
			// NOTE Duplicated Operation.Hash, the latter will be ignored.
			if _, found := founds[op.Operation()]; found {
				continue
			} else if found, err := pp.localstate.Storage().HasOperation(op.Operation()); err != nil {
				return nil, err
			} else if found { // already stored Operation
				continue
			}

			operations = append(operations, op)
			founds[op.Operation()] = struct{}{}
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

func (pp *proposalProcessorV0) processStates() (*tree.AVLTree, error) {
	pool := NewStatePool(pp.localstate.Storage())

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

		if st, err := opp.ProcessOperation(pool.Get, pool.Set); err != nil {
			pp.Log().Error().Err(err).
				Interface("operation", op).
				Msg("failed to process operation")
			continue
		} else if st != nil {
			if err := st.SetPreviousBlock(pp.lastManifest.Hash()); err != nil {
				return nil, err
			}
			if err := st.AddOperationInfo(opi); err != nil {
				return nil, err
			}
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

func (pp *proposalProcessorV0) getOperationsFromStorage(h valuehash.Hash) ([]state.OperationInfoV0, error) {
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
		if _, found := pp.proposedOperations[op.Hash()]; !found {
			continue
		}

		ops[i] = state.NewOperationInfoV0(op, h)
	}

	return ops, nil
}

func (pp *proposalProcessorV0) getOperationsThruChannel(
	proposer base.Address,
	notFounds []valuehash.Hash,
	founds map[valuehash.Hash]state.OperationInfoV0,
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
				if _, found := pp.proposedOperations[op.Hash()]; !found {
					continue
				} else if _, found := founds[op.Hash()]; found {
					continue
				}

				ops = append(ops, state.NewOperationInfoV0(op, sl.Hash()))
			}
		}
	}

	return ops, nil
}

func (pp *proposalProcessorV0) setACCEPTVoteproof(acceptVoteproof base.Voteproof) error {
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

func (pp *proposalProcessorV0) validateTree(tg *avl.TreeGenerator) (*tree.AVLTree, error) {
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

func (pp *proposalProcessorV0) updateStates(tr *tree.AVLTree, blk block.Block) error {
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
