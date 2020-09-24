package isaac

import (
	"bytes"
	"context"
	"sort"
	"sync"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/tree"
	"github.com/spikeekips/mitum/util/valuehash"
)

type internalDefaultProposalProcessor struct {
	sync.RWMutex
	*logging.Logging
	localstate   *Localstate
	stopped      bool
	suffrage     base.Suffrage
	lastManifest block.Manifest
	block        block.BlockUpdater
	proposal     ballot.Proposal
	operations   []operation.Operation
	bs           storage.BlockStorage
	si           block.SuffrageInfoV0
	oprHintset   *hint.Hintmap
	statesValue  *sync.Map
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
		localstate:   localstate,
		suffrage:     suffrage,
		proposal:     proposal,
		lastManifest: lastManifest,
		si:           si,
		oprHintset:   oprHintset,
		statesValue:  &sync.Map{},
	}, nil
}

func (pp *internalDefaultProposalProcessor) stop() {
	pp.Lock()
	defer pp.Unlock()

	pp.stopped = true
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
		pp.statesValue.Store("process", time.Since(s))

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
		return nil, xerrors.Errorf("timeout to process Proposal: %w", err)
	case blk = <-blkChan:
		if err := pp.setBlockfs(blk); err != nil {
			return nil, xerrors.Errorf("failed to set blockfs: %w", err)
		}
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

	var pool *Statepool
	if p, err := NewStatepool(pp.localstate.Storage()); err != nil {
		return nil, err
	} else {
		pool = p
	}

	var operationsTree, statesTree tree.FixedTree
	var sts []state.State
	if len(pp.operations) > 0 {
		var err error
		if statesTree, sts, err = pp.processStates(ctx, pool); err != nil {
			return nil, err
		} else if !statesTree.IsEmpty() {
			if operationsTree, err = pp.processOperations(pool); err != nil {
				return nil, err
			}
		}
	}

	return pp.createBlock(initVoteproof, operationsTree, statesTree, sts)
}

func (pp *internalDefaultProposalProcessor) createBlock(
	initVoteproof base.Voteproof,
	operationsTree, statesTree tree.FixedTree,
	sts []state.State,
) (block.Block, error) {
	var opsHash, stsHash valuehash.Hash
	if !operationsTree.IsEmpty() {
		opsHash = valuehash.NewBytes(operationsTree.Root())
	}
	if !statesTree.IsEmpty() {
		stsHash = valuehash.NewBytes(statesTree.Root())
	}

	var blk block.BlockUpdater
	if b, err := block.NewBlockV0(
		pp.si, pp.proposal.Height(), pp.proposal.Round(), pp.proposal.Hash(), pp.lastManifest.Hash(),
		opsHash, stsHash, pp.proposal.SignedAt(),
	); err != nil {
		return nil, err
	} else {
		blk = b
	}

	blk = blk.SetOperationsTree(operationsTree).SetOperations(pp.operations).
		SetStatesTree(statesTree).SetStates(sts).
		SetINITVoteproof(initVoteproof).SetProposal(pp.proposal)

	if bs, err := pp.localstate.Storage().OpenBlockStorage(blk); err != nil {
		return nil, err
	} else {
		pp.bs = bs
		pp.block = blk
	}

	pp.Log().Debug().
		Dict("block", logging.Dict().
			Hinted("hash", blk.Hash()).Hinted("height", blk.Height()).Hinted("round", blk.Round()).
			Hinted("proposal", blk.Proposal()).Hinted("previous_block", blk.PreviousBlock()).
			Hinted("operations_hash", blk.OperationsHash()).Hinted("states_hash", blk.StatesHash()),
		).Msg("block processed")

	return blk, nil
}

func (pp *internalDefaultProposalProcessor) extractOperations() ([]operation.Operation, error) {
	s := time.Now()
	defer func() {
		pp.statesValue.Store("extract-operations", time.Since(s))
	}()

	if pp.isStopped() {
		return nil, xerrors.Errorf("already stopped")
	}

	founds := map[string]struct{}{}
	bySeals := map[string][]operation.Operation{}
	var notFounds []valuehash.Hash
	for _, h := range pp.proposal.Seals() {
		switch l, found, err := pp.getOperationsFromStorage(h); {
		case err != nil:
			return nil, err
		case !found:
			notFounds = append(notFounds, h)

			continue
		default:
			ops := pp.filterOps(l, founds)
			if len(ops) < 1 {
				continue
			}

			bySeals[h.String()] = ops
		}
	}

	if len(notFounds) > 0 {
		if m, err := pp.getOperationsThruChannel(pp.proposal.Node(), notFounds, founds); err != nil {
			return nil, err
		} else {
			for k := range m {
				bySeals[k] = m[k]
			}
		}
	}

	var ops []operation.Operation
	for _, h := range pp.proposal.Seals() {
		if l, found := bySeals[h.String()]; !found {
			continue
		} else {
			ops = append(ops, l...)
		}
	}

	return ops, nil
}

func (pp *internalDefaultProposalProcessor) filterOps(
	ops []operation.Operation,
	founds map[string]struct{},
) []operation.Operation {
	if len(ops) < 1 {
		return nil
	}

	var nops []operation.Operation
	for i := range ops {
		op := ops[i]
		fk := op.Hash().String()
		if _, found := founds[fk]; found {
			continue
		} else {
			nops = append(nops, op)
			founds[fk] = struct{}{}
		}
	}

	return nops
}

func (pp *internalDefaultProposalProcessor) processStates(
	ctx context.Context,
	pool *Statepool,
) (tree.FixedTree, []state.State, error) {
	s := time.Now()
	defer func() {
		pp.statesValue.Store("process-states", time.Since(s))
	}()

	var co *ConcurrentOperationsProcessor
	if c, err := NewConcurrentOperationsProcessor(len(pp.operations), pool, pp.oprHintset); err != nil {
		return tree.FixedTree{}, nil, err
	} else {
		_ = c.SetLogger(pp.Log())

		co = c.Start(
			ctx,
			func(sp state.Processor) error {
				switch found, err := pp.localstate.Storage().HasOperationFact(sp.(operation.Operation).Fact().Hash()); {
				case err != nil:
					return err
				case found:
					return state.IgnoreOperationProcessingError.Errorf("already known")
				default:
					return nil
				}
			},
		)

		go func() {
			<-ctx.Done()
			_ = co.Cancel()
		}()
	}

	for i := range pp.operations {
		op := pp.operations[i]

		pp.Log().Verbose().Hinted("fact", op.Fact().Hash()).Msg("process fact")

		if pp.isStopped() {
			return tree.FixedTree{}, nil, xerrors.Errorf("already stopped")
		} else if err := co.Process(op); err != nil {
			return tree.FixedTree{}, nil, err
		}
	}

	if err := co.Close(); err != nil {
		return tree.FixedTree{}, nil, err
	}

	if !pool.IsUpdated() {
		return tree.FixedTree{}, nil, nil
	}

	return pp.generateStatesTree(pool)
}

func (pp *internalDefaultProposalProcessor) processOperations(pool *Statepool) (tree.FixedTree, error) {
	s := time.Now()
	defer func() {
		pp.statesValue.Store("process-operations", time.Since(s))
	}()

	statesOps := pool.InsertedOperations()
	for _, op := range pool.AddedOperations() {
		pp.operations = append(pp.operations, op)
	}

	tg := tree.NewFixedTreeGenerator(uint(len(pp.operations)), nil)
	for i := range pp.operations {
		if pp.isStopped() {
			return tree.FixedTree{}, xerrors.Errorf("already stopped")
		}

		fh := pp.operations[i].Fact().Hash()

		var mod []byte
		if _, found := statesOps[fh.String()]; found {
			mod = base.FactMode2bytes(base.FInStates)
		}

		if err := tg.Add(i, fh.Bytes(), mod); err != nil {
			return tree.FixedTree{}, err
		}
	}

	if tr, err := tg.Tree(); err != nil {
		return tree.FixedTree{}, err
	} else {
		return tr, nil
	}
}

func (pp *internalDefaultProposalProcessor) generateStatesTree(pool *Statepool) (tree.FixedTree, []state.State, error) {
	sts := make([]state.State, len(pool.Updates()))
	for i, s := range pool.Updates() {
		if pp.isStopped() {
			return tree.FixedTree{}, nil, xerrors.Errorf("already stopped")
		}

		st := s.GetState()
		if ust, err := st.SetHash(st.GenerateHash()); err != nil {
			return tree.FixedTree{}, nil, err
		} else if err := ust.IsValid(nil); err != nil {
			return tree.FixedTree{}, nil, err
		} else {
			sts[i] = ust
		}
	}

	sort.Slice(sts, func(i, j int) bool {
		return bytes.Compare(sts[i].Hash().Bytes(), sts[j].Hash().Bytes()) < 0
	})

	tg := tree.NewFixedTreeGenerator(uint(len(pool.Updates())), nil)
	for i := range sts {
		if err := tg.Add(i, sts[i].Hash().Bytes(), nil); err != nil {
			return tree.FixedTree{}, nil, err
		}
	}

	if tr, err := tg.Tree(); err != nil {
		return tree.FixedTree{}, nil, err
	} else {
		return tr, sts, nil
	}
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

	var ops []operation.Operation // nolint
	for i := range osl.Operations() {
		if pp.isStopped() {
			return nil, false, xerrors.Errorf("already stopped")
		}

		ops = append(ops, osl.Operations()[i])
	}

	return ops, true, nil
}

func (pp *internalDefaultProposalProcessor) getOperationsThruChannel(
	proposer base.Address,
	notFounds []valuehash.Hash,
	founds map[string]struct{},
) (map[string][]operation.Operation, error) {
	if pp.isStopped() {
		return nil, xerrors.Errorf("already stopped")
	}

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

	bySeals := map[string][]operation.Operation{}
	for i := range received {
		sl := received[i]
		if os, ok := sl.(operation.Seal); !ok {
			return nil, xerrors.Errorf("not operation.Seal: %T", sl)
		} else {
			l := pp.filterOps(os.Operations(), founds)
			bySeals[sl.Hash().String()] = l
		}
	}

	return bySeals, nil
}

func (pp *internalDefaultProposalProcessor) setACCEPTVoteproof(acceptVoteproof base.Voteproof) error {
	pp.Lock()
	defer pp.Unlock()

	if pp.bs == nil {
		return xerrors.Errorf("not yet processed")
	}

	s := time.Now()
	defer func() {
		pp.statesValue.Store("set-accept-voteproof", time.Since(s))
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

	blk := pp.block.SetACCEPTVoteproof(acceptVoteproof)

	if err := func() error {
		s := time.Now()
		defer func() {
			pp.statesValue.Store("set-block", time.Since(s))
		}()

		return pp.bs.SetBlock(blk)
	}(); err != nil {
		return err
	}

	if err := pp.localstate.BlockFS().AddACCEPTVoteproof(blk.Height(), blk.Hash(), acceptVoteproof); err != nil {
		return err
	}

	if seals := pp.proposal.Seals(); len(seals) > 0 {
		// TODO when failed, seals of UnstageOperationSeals should be recovered
		if err := func() error {
			s := time.Now()
			defer func() {
				pp.statesValue.Store("unstage-operation-seals", time.Since(s))
			}()

			return pp.bs.UnstageOperationSeals(seals) // TODO if failed to commit, seals should be recovered
		}(); err != nil {
			return err
		}
	}
	pp.block = blk

	return nil
}

func (pp *internalDefaultProposalProcessor) states() map[string]interface{} {
	m := map[string]interface{}{}
	pp.statesValue.Range(func(key, value interface{}) bool {
		m[key.(string)] = value

		return true
	})

	return m
}

func (pp *internalDefaultProposalProcessor) setBlockfs(blk block.Block) error {
	s := time.Now()
	defer func() {
		pp.statesValue.Store("blockfs", time.Since(s))
	}()

	return pp.localstate.BlockFS().Add(blk)
}
