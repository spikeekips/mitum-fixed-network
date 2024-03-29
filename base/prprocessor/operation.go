package prprocessor

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/tree"
	"github.com/spikeekips/mitum/util/valuehash"
)

var maxConcurrentOperations int64 = 500

type OperationProcessor interface {
	New(*storage.Statepool) OperationProcessor
	PreProcess(state.Processor) (state.Processor, error)
	Process(state.Processor) error
	Close() error
	Cancel() error
}

type defaultOperationProcessor struct {
	pool *storage.Statepool
}

func (defaultOperationProcessor) New(pool *storage.Statepool) OperationProcessor {
	return &defaultOperationProcessor{
		pool: pool,
	}
}

func (opp defaultOperationProcessor) PreProcess(op state.Processor) (state.Processor, error) {
	pr, ok := op.(state.PreProcessor)
	if ok {
		return pr.PreProcess(opp.pool.Get, opp.pool.Set)
	}

	return op, nil
}

func (opp defaultOperationProcessor) Process(op state.Processor) error {
	return op.Process(opp.pool.Get, opp.pool.Set)
}

func (defaultOperationProcessor) Close() error {
	return nil
}

func (defaultOperationProcessor) Cancel() error {
	return nil
}

type ConcurrentOperationsProcessor struct {
	sync.RWMutex
	*logging.Logging
	max              int64
	pool             *storage.Statepool
	wk               *util.ErrgroupWorker
	donechan         chan error
	oprLock          sync.RWMutex
	oppHintSet       *hint.Hintmap
	oprs             map[hint.Hint]OperationProcessor
	workFilter       func(state.Processor) error
	closed           bool
	opsTreeGenerator *tree.FixedTreeGenerator
}

func NewConcurrentOperationsProcessor(
	size uint64,
	max int64,
	pool *storage.Statepool,
	oppHintSet *hint.Hintmap,
) (*ConcurrentOperationsProcessor, error) {
	if max < 1 {
		return nil, errors.Errorf("max must be over 0")
	} else if max > maxConcurrentOperations {
		max = maxConcurrentOperations
	}

	return &ConcurrentOperationsProcessor{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "concurrent-operations-processor")
		}),
		max:        max,
		pool:       pool,
		oppHintSet: oppHintSet,

		oprs:             map[hint.Hint]OperationProcessor{},
		workFilter:       func(state.Processor) error { return nil },
		opsTreeGenerator: tree.NewFixedTreeGenerator(size),
	}, nil
}

func (co *ConcurrentOperationsProcessor) addOperationsTree(index uint64, fact valuehash.Hash, reason error) error {
	no := operation.NewFixedTreeNode(index, fact.Bytes(), reason == nil, reason)

	return co.opsTreeGenerator.Add(no)
}

func (co *ConcurrentOperationsProcessor) Start(
	ctx context.Context,
	workFilter func(state.Processor) error,
) *ConcurrentOperationsProcessor {
	if workFilter != nil {
		co.workFilter = workFilter
	}

	co.wk = util.NewErrgroupWorker(ctx, co.max)

	co.donechan = make(chan error, 2)
	go func() {
		defer co.wk.Close()

		co.donechan <- co.wk.Wait()
	}()

	return co
}

func (co *ConcurrentOperationsProcessor) StatesTree() (tree.FixedTree, []state.State, error) {
	co.RLock()
	defer co.RUnlock()

	updates := co.pool.Updates()
	states := make([]state.State, len(updates))
	for i := range updates {
		s := updates[i]
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

	trg := tree.NewFixedTreeGenerator(uint64(len(updates)))
	for i := range states {
		if err := trg.Add(state.NewFixedTreeNode(uint64(i), states[i].Hash().Bytes())); err != nil {
			return tree.FixedTree{}, nil, err
		}
	}

	tr, err := trg.Tree()
	if err != nil {
		return tree.FixedTree{}, nil, err
	}

	return tr, states, nil
}

func (co *ConcurrentOperationsProcessor) OperationsTree() (tree.FixedTree, error) {
	co.RLock()
	defer co.RUnlock()

	added := co.pool.AddedOperations()
	size := uint64(co.opsTreeGenerator.Len())
	n := len(added)
	if n < 1 {
		return co.opsTreeGenerator.Tree()
	}

	trg := tree.NewFixedTreeGenerator(size + uint64(n))
	if err := co.opsTreeGenerator.Traverse(func(no tree.FixedTreeNode) (bool, error) {
		if err := trg.Add(no); err != nil {
			return false, err
		}

		return true, nil
	}); err != nil {
		return tree.FixedTree{}, err
	}

	co.opsTreeGenerator = trg

	var i uint64
	for k := range added {
		op := added[k]
		if err := co.addOperationsTree(size+i, op.Fact().Hash(), nil); err != nil {
			return tree.FixedTree{}, err
		}
		i++
	}

	return co.opsTreeGenerator.Tree()
}

func (co *ConcurrentOperationsProcessor) Process(index uint64, op operation.Operation) error {
	if co.wk == nil {
		return errors.Errorf("not started")
	}

	l := co.Log().With().
		Stringer("operation", op.Hash()).Stringer("fact", op.Fact().Hash()).Logger()

	l.Trace().Func(func(e *zerolog.Event) {
		e.Interface("operation", op)
	}).Msg("operation will be processed")

	if pr, ok := op.(state.Processor); !ok {
		l.Trace().Msgf("not state.StateProcessor, %T", op)

		return co.addOperationsTree(
			index,
			op.Fact().Hash(),
			operation.NewBaseReasonError("not operation, %T", op),
		)
	} else if err := co.process(index, pr); err != nil {
		if err0 := co.addOperationsTree(index, op.Fact().Hash(), err); err0 != nil {
			return err0
		}

		if operationIgnored(err) {
			l.Trace().Err(err).Msg("operation ignored")

			return nil
		}

		l.Trace().Err(err).Msg("operation failed to PreProcess")

		return err
	}

	l.Trace().Msg("operation ready to process")

	return nil
}

func (co *ConcurrentOperationsProcessor) process(index uint64, op state.Processor) error {
	if opr, err := co.opr(op); err != nil {
		return err
	} else if ppr, err := opr.PreProcess(op); err != nil {
		return err
	} else if err := co.wk.NewJob(func(context.Context, uint64) error {
		return co.work(index, ppr)
	}); err != nil {
		return util.IgnoreError.Errorf("operation processor already closed")
	}

	return nil
}

func (co *ConcurrentOperationsProcessor) Cancel() error {
	co.Lock()
	defer co.Unlock()

	if co.wk == nil || co.closed {
		return nil
	}

	co.wk.LazyCancel(time.Second)
	co.closed = true

	errchan := make(chan error, len(co.oprs))

	var wg sync.WaitGroup
	wg.Add(len(co.oprs))
	for _, opr := range co.oprs {
		opr := opr
		go func() {
			defer wg.Done()

			errchan <- opr.Cancel()
		}()
	}

	wg.Wait()
	close(errchan)

	for err := range errchan {
		if err != nil {
			return err
		}
	}

	return nil
}

func (co *ConcurrentOperationsProcessor) Close() error {
	co.Lock()
	defer co.Unlock()

	if co.wk == nil || co.closed {
		return nil
	}

	co.wk.Done()

	co.closed = true

	if err := <-co.donechan; err != nil {
		return err
	}

	errchan := make(chan error, len(co.oprs))

	var wg sync.WaitGroup
	wg.Add(len(co.oprs))
	for _, opr := range co.oprs {
		opr := opr
		go func() {
			defer wg.Done()

			errchan <- opr.Close()
		}()
	}

	wg.Wait()
	close(errchan)

	for err := range errchan {
		if err != nil {
			return err
		}
	}

	return nil
}

func (co *ConcurrentOperationsProcessor) opr(op state.Processor) (OperationProcessor, error) {
	co.oprLock.Lock()
	defer co.oprLock.Unlock()

	hinter, ok := op.(hint.Hinter)
	if !ok {
		return nil, errors.Errorf("not Hinter, %T", op)
	}

	if opr, found := co.oprs[hinter.Hint()]; found {
		return opr, nil
	}

	var opr OperationProcessor = defaultOperationProcessor{}
	if co.oppHintSet != nil {
		if hinter, err := co.oppHintSet.Compatible(hinter); err != nil {
			opr = defaultOperationProcessor{}
		} else {
			opr = hinter.(OperationProcessor)
		}
	}

	opr = opr.New(co.pool)
	co.oprs[hinter.Hint()] = opr

	return opr, nil
}

func (co *ConcurrentOperationsProcessor) work(jobid uint64, ppr state.Processor) error {
	op, ok := ppr.(operation.Operation)
	if !ok {
		return fmt.Errorf("operation.Operation is not operation.Operation, %T", ppr)
	}

	err := co.workProcess(ppr)

	if cerr := co.addOperationsTree(jobid, op.Fact().Hash(), err); cerr != nil {
		return cerr
	}

	if operationIgnored(err) {
		return nil
	}

	return err
}

func (co *ConcurrentOperationsProcessor) workProcess(op state.Processor) error {
	if err := co.workFilter(op); err != nil {
		return err
	}

	opr, err := co.opr(op)
	if err != nil {
		return err
	}

	if err = opr.Process(op); err != nil {
		co.Log().Trace().
			Stringer("operation", op.(operation.Operation).Hash()).
			Err(err).
			Msg("operation failed to process")

		return err
	}

	co.Log().Trace().Stringer("operation", op.(operation.Operation).Hash()).Err(err).Msg("operation processed")

	return nil
}

func operationIgnored(err error) bool {
	if err == nil {
		return false
	}

	var operr operation.ReasonError
	switch {
	case errors.Is(err, util.IgnoreError):
		return true
	case errors.As(err, &operr):
		return true
	default:
		return false
	}
}
