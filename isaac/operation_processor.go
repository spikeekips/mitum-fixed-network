package isaac

import (
	"context"
	"sync"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
)

var maxConcurrentOperations int = 500

type OperationProcessor interface {
	New(*Statepool) OperationProcessor
	PreProcess(state.Processor) (state.Processor, error)
	Process(state.Processor) error
	Close() error
	Cancel() error
}

type defaultOperationProcessor struct {
	pool *Statepool
}

func (opp defaultOperationProcessor) New(pool *Statepool) OperationProcessor {
	return &defaultOperationProcessor{
		pool: pool,
	}
}

func (opp defaultOperationProcessor) PreProcess(op state.Processor) (state.Processor, error) {
	if pr, ok := op.(state.PreProcessor); ok {
		return pr.PreProcess(opp.pool.Get, opp.pool.Set)
	} else {
		return op, nil
	}
}

func (opp defaultOperationProcessor) Process(op state.Processor) error {
	return op.Process(opp.pool.Get, opp.pool.Set)
}

func (opp defaultOperationProcessor) Close() error {
	return nil
}

func (opp defaultOperationProcessor) Cancel() error {
	return nil
}

type ConcurrentOperationsProcessor struct {
	sync.RWMutex
	*logging.Logging
	size       uint
	pool       *Statepool
	wk         *util.DistributeWorker
	donechan   chan error
	oprLock    sync.RWMutex
	oppHintSet *hint.Hintmap
	oprs       map[hint.Hint]OperationProcessor
	workFilter func(state.Processor) error
	closed     bool
}

func NewConcurrentOperationsProcessor(
	size int,
	pool *Statepool,
	oppHintSet *hint.Hintmap,
) (*ConcurrentOperationsProcessor, error) {
	if size < 1 {
		return nil, xerrors.Errorf("size must be over 0")
	} else if size > maxConcurrentOperations {
		size = maxConcurrentOperations
	}

	return &ConcurrentOperationsProcessor{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "concurrent-operations-processor")
		}),
		size:       uint(size),
		pool:       pool,
		oppHintSet: oppHintSet,
		oprs:       map[hint.Hint]OperationProcessor{},
		workFilter: func(state.Processor) error { return nil },
	}, nil
}

func (co *ConcurrentOperationsProcessor) Start(
	ctx context.Context,
	workFilter func(state.Processor) error,
) *ConcurrentOperationsProcessor {
	if workFilter != nil {
		co.workFilter = workFilter
	}

	errchan := make(chan error)
	co.wk = util.NewDistributeWorker(co.size, errchan)

	co.donechan = make(chan error, 2)
	go func() {
		<-ctx.Done()
		co.donechan <- xerrors.Errorf("canceled to process: %w", ctx.Err())
	}()

	go func() {
		if err := co.wk.Run(co.work); err != nil {
			errchan <- err
		}

		close(errchan)
	}()

	go func() {
		for err := range errchan {
			if err == nil || xerrors.Is(err, util.IgnoreError) {
				continue
			}

			co.wk.Done(false)
			co.donechan <- err

			return
		}

		co.donechan <- nil
	}()

	return co
}

func (co *ConcurrentOperationsProcessor) Process(op operation.Operation) error {
	if co.wk == nil {
		return xerrors.Errorf("not started")
	}

	l := co.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("operation", op.Hash()).Hinted("fact", op.Fact().Hash())
	})

	l.Verbose().Msg("opertion will be processed")

	if pr, ok := op.(state.Processor); !ok {
		l.Verbose().Msgf("not state.StateProcessor, %T", op)

		return nil
	} else if err := co.process(pr); err != nil {
		if xerrors.Is(err, util.IgnoreError) {
			l.Verbose().Err(err).Msg("operation ignored")

			return nil
		}

		l.Verbose().Err(err).Msg("operation failed to PreProcess")

		return err
	}

	l.Verbose().Msg("operation ready to process")

	return nil
}

func (co *ConcurrentOperationsProcessor) process(op state.Processor) error {
	if opr, err := co.opr(op); err != nil {
		return err
	} else if ppr, err := opr.PreProcess(op); err != nil {
		return err
	} else if !co.wk.NewJob(ppr) {
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

	co.wk.Done(false)
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

	co.wk.Done(true)
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

	var hinter hint.Hinter
	if ht, ok := op.(hint.Hinter); !ok {
		return nil, xerrors.Errorf("not hint.Hinter, %T", op)
	} else {
		hinter = ht
	}

	if opr, found := co.oprs[hinter.Hint()]; found {
		return opr, nil
	}

	var opr OperationProcessor
	if hinter, found := co.oppHintSet.Get(hinter); !found {
		opr = defaultOperationProcessor{}
	} else {
		opr = hinter.(OperationProcessor)
	}

	opr = opr.New(co.pool)
	co.oprs[hinter.Hint()] = opr

	if l, ok := opr.(logging.SetLogger); ok {
		_ = l.SetLogger(co.Log())
	}

	return opr, nil
}

func (co *ConcurrentOperationsProcessor) work(_ uint, j interface{}) error {
	if j == nil {
		return nil
	}

	var op state.Processor
	if sp, ok := j.(state.Processor); !ok {
		return util.IgnoreError.Errorf("not state.StateProcessor, %T", j)
	} else {
		op = sp
	}

	if err := co.workFilter(op); err != nil {
		return err
	}

	if opr, err := co.opr(op); err != nil {
		return err
	} else {
		if err := opr.Process(op); err != nil {
			co.Log().Verbose().
				Hinted("operation", op.(operation.Operation).Hash()).
				Err(err).
				Msg("operation failed to process")

			return err
		} else {
			co.Log().Verbose().Hinted("operation", op.(operation.Operation).Hash()).Err(err).Msg("operation processed")

			return nil
		}
	}
}
