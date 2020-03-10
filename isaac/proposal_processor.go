package isaac

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/avl"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/operation"
	"github.com/spikeekips/mitum/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/tree"
	"github.com/spikeekips/mitum/valuehash"
)

type ProposalProcessor interface {
	ProcessINIT(valuehash.Hash /* Proposal.Hash() */, Voteproof /* INIT Voteproof */) (Block, error)
	ProcessACCEPT(valuehash.Hash /* Proposal.Hash() */, Voteproof /* ACCEPT Voteproof */) (BlockStorage, error)
}

type ProposalProcessorV0 struct {
	*logging.Logger
	localstate *Localstate
	processors *sync.Map
}

func NewProposalProcessorV0(localstate *Localstate) *ProposalProcessorV0 {
	return &ProposalProcessorV0{
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "proposal-processor-v0")
		}),
		localstate: localstate,
		processors: &sync.Map{},
	}
}

func (dp *ProposalProcessorV0) ProcessINIT(ph valuehash.Hash, initVoteproof Voteproof) (Block, error) {
	if i, found := dp.processors.Load(ph); found {
		processor := i.(*proposalProcessorV0)

		return processor.block, nil
	}

	if initVoteproof.Stage() != StageINIT {
		return nil, xerrors.Errorf("ProcessINIT needs INIT Voteproof")
	}

	var proposal Proposal
	if sl, err := dp.localstate.Storage().Seal(ph); err != nil {
		return nil, err
	} else if pr, ok := sl.(Proposal); !ok {
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

	processor, err := newProposalProcessorV0(dp.localstate, proposal)
	if err != nil {
		return nil, err
	}

	_ = processor.SetLogger(*dp.Log())

	block, err := processor.processINIT(initVoteproof)
	if err != nil {
		return nil, err
	}

	dp.processors.Store(ph, processor)

	return block, nil
}

func (dp *ProposalProcessorV0) ProcessACCEPT(
	ph valuehash.Hash, acceptVoteproof Voteproof,
) (BlockStorage, error) {
	if acceptVoteproof.Stage() != StageACCEPT {
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

	return processor.bs, nil
}

type proposalProcessorV0 struct {
	*logging.Logger
	localstate *Localstate
	lastBlock  Block
	block      Block
	proposal   Proposal
	operations []state.OperationInfoV0
	bs         BlockStorage
}

func newProposalProcessorV0(localstate *Localstate, proposal Proposal) (*proposalProcessorV0, error) {
	lastBlock := localstate.LastBlock()
	if lastBlock == nil {
		return nil, xerrors.Errorf("last block is empty")
	}

	return &proposalProcessorV0{
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "internal-proposal-processor-inside-v0")
		}),
		localstate: localstate,
		proposal:   proposal,
		lastBlock:  lastBlock,
	}, nil
}

func (pp *proposalProcessorV0) processINIT(initVoteproof Voteproof) (Block, error) {
	if pp.block != nil {
		return pp.block, nil
	}

	blockOperations, operationTree, err := pp.processOperations()
	if err != nil {
		return nil, err
	}

	blockStates, stateTree, err := pp.processStates()
	if err != nil {
		return nil, err
	}

	var block Block
	if b, err := NewBlockV0(
		pp.proposal.Height(), pp.proposal.Round(), pp.proposal.Hash(), pp.lastBlock.Hash(),
		blockOperations, blockStates,
		pp.localstate.Policy().NetworkID(),
	); err != nil {
		return nil, err
	} else {
		block = b.SetINITVoteproof(initVoteproof)
	}

	if bs, err := pp.localstate.Storage().OpenBlockStorage(block); err != nil {
		return nil, err
	} else {
		pp.bs = bs
	}

	if operationTree != nil {
		if err := pp.bs.SetOperations(operationTree); err != nil {
			return nil, err
		}
	}

	if stateTree != nil {
		if err := pp.bs.SetStates(stateTree); err != nil {
			return nil, err
		}
	}

	pp.block = block

	return block, nil
}

func (pp *proposalProcessorV0) extractOperations() ([]state.OperationInfoV0, error) {
	// NOTE the order of operation should be kept by the order of
	// Proposal.Seals()
	seals := map[valuehash.Hash][]state.OperationInfoV0{}

	var notFounds []valuehash.Hash
	for _, h := range pp.proposal.Seals() {
		if ops, err := pp.getOperationsFromStorage(h); err != nil {
			if xerrors.Is(err, storage.NotFoundError) {
				notFounds = append(notFounds, h)
				continue
			}

			return nil, err
		} else {
			seals[h] = ops
		}
	}

	if len(notFounds) > 0 {
		if sos, err := pp.getOperationsThruChannel(pp.proposal.Node(), notFounds); err != nil {
			return nil, err
		} else {
			for h := range sos {
				seals[h] = sos[h]
			}
		}
	}

	var operations []state.OperationInfoV0 // nolint
	for _, h := range pp.proposal.Seals() {
		operations = append(operations, seals[h]...)
	}

	return operations, nil
}

func (pp *proposalProcessorV0) processOperations() (valuehash.Hash, *tree.AVLTree, error) {
	if len(pp.proposal.Seals()) < 1 {
		return nil, nil, nil
	}

	// TODO operations should not be duplicated
	var operations []state.OperationInfoV0

	if ops, err := pp.extractOperations(); err != nil {
		return nil, nil, err
	} else {
		founds := map[valuehash.Hash]struct{}{}

		// NOTE check the duplication of Operation.Hash. If found, the latter
		// will be ignored.
		for i := range ops {
			op := ops[i]
			if _, found := founds[op.Operation()]; found {
				continue
			}
			operations = append(operations, op)
			founds[op.Operation()] = struct{}{}
		}
	}

	if len(operations) < 1 {
		return nil, nil, nil
	}

	tg := avl.NewTreeGenerator()
	for i := range operations {
		op := operations[i]
		n := operation.NewOperationAVLNode(op.RawOperation())
		if _, err := tg.Add(n); err != nil {
			return nil, nil, err
		}
	}

	boh, tr, err := pp.validateTree(tg)
	if err != nil {
		return nil, nil, err
	}

	pp.operations = operations

	return boh, tr, nil
}

func (pp *proposalProcessorV0) processStates() (valuehash.Hash, *tree.AVLTree, error) {
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
			if err := st.SetPreviousBlock(pp.lastBlock.Hash()); err != nil {
				return nil, nil, err
			}
			if err := st.AddOperationInfo(opi); err != nil {
				return nil, nil, err
			}
		}
	}

	updated := pool.Updated()
	if len(updated) < 1 {
		return nil, nil, nil
	}

	tg := avl.NewTreeGenerator()
	for _, s := range updated {
		if err := s.SetHash(s.GenerateHash()); err != nil {
			return nil, nil, err
		}

		if err := s.IsValid(nil); err != nil {
			return nil, nil, err
		}

		sv := s.(*state.StateV0)
		n := state.NewStateV0AVLNode(*sv)
		if _, err := tg.Add(n); err != nil {
			return nil, nil, err
		}
	}

	boh, tr, err := pp.validateTree(tg)
	if err != nil {
		return nil, nil, err
	}

	return boh, tr, nil
}

func (pp *proposalProcessorV0) getOperationsFromStorage(h valuehash.Hash) ([]state.OperationInfoV0, error) {
	var osl operation.Seal
	if sl, err := pp.localstate.Storage().Seal(h); err != nil {
		return nil, err
	} else if os, ok := sl.(operation.Seal); !ok {
		return nil, xerrors.Errorf("not operation.Seal: %T", sl)
	} else {
		osl = os
	}

	var ops []state.OperationInfoV0 // nolint
	for _, op := range osl.Operations() {
		ops = append(ops, state.NewOperationInfoV0(op, h))
	}

	return ops, nil
}

func (pp *proposalProcessorV0) getOperationsThruChannel(
	proposer Address,
	notFounds []valuehash.Hash,
) (map[valuehash.Hash][]state.OperationInfoV0, error) {
	if pp.localstate.Node().Address().Equal(proposer) {
		pp.Log().Warn().Msg("proposer is local node, but local node should have seals. Hmmm")
	}

	node, found := pp.localstate.Nodes().Node(proposer)
	if !found {
		return nil, xerrors.Errorf("unknown proposer: %v", proposer)
	}

	sos := map[valuehash.Hash][]state.OperationInfoV0{}
	received, err := node.Channel().Seals(notFounds)
	if err != nil {
		return nil, err
	}

	for _, sl := range received {
		if err := pp.localstate.Storage().NewSeal(sl); err != nil {
			return nil, err
		}

		if os, ok := sl.(operation.Seal); !ok {
			return nil, xerrors.Errorf("not operation.Seal: %T", sl)
		} else {
			var ops []state.OperationInfoV0
			for _, op := range os.Operations() {
				ops = append(ops, state.NewOperationInfoV0(op, sl.Hash()))
			}

			sos[sl.Hash()] = ops
		}
	}

	return sos, nil
}

func (pp *proposalProcessorV0) setACCEPTVoteproof(acceptVoteproof Voteproof) error {
	if pp.bs == nil {
		return xerrors.Errorf("not yet processed")
	}

	block := pp.block.SetACCEPTVoteproof(acceptVoteproof)
	if err := pp.bs.SetBlock(block); err != nil {
		return err
	}
	pp.block = block

	return nil
}

func (pp *proposalProcessorV0) validateTree(tg *avl.TreeGenerator) (valuehash.Hash, *tree.AVLTree, error) {
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

	var rootHash valuehash.Hash
	if h, err := valuehash.LoadSHA256FromBytes(tr.Root().Hash()); err != nil {
		return nil, nil, err
	} else {
		rootHash = h
	}

	return rootHash, tr, nil
}
