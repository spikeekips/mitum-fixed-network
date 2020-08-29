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

type ConcurrentOperationsProcessor struct {
	*logging.Logging
	size       uint
	pool       *Statepool
	wk         *util.DistributeWorker
	donechan   chan error
	oprLock    sync.RWMutex
	oppHintSet *hint.Hintmap
	oprs       map[hint.Hint]OperationProcessor
	localstate *Localstate
}

func NewConcurrentOperationsProcessor(
	localstate *Localstate,
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
		localstate: localstate,
		size:       uint(size),
		pool:       pool,
		oppHintSet: oppHintSet,
		oprs:       map[hint.Hint]OperationProcessor{},
	}, nil
}

func (co *ConcurrentOperationsProcessor) Start(ctx context.Context) *ConcurrentOperationsProcessor {
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
			if err == nil || xerrors.Is(err, state.IgnoreOperationProcessingError) {
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
		return ctx.Hinted("operation", op.Hash())
	})

	if pr, ok := op.(state.Processor); !ok {
		co.Log().Verbose().Msgf("not state.StateProcessor, %T", op)

		return nil
	} else if err := co.process(pr); err != nil {
		if xerrors.Is(err, state.IgnoreOperationProcessingError) {
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
		return state.IgnoreOperationProcessingError.Errorf("already closed")
	}

	return nil
}

func (co *ConcurrentOperationsProcessor) Cancel() error {
	if co.wk == nil {
		return nil
	}

	co.wk.Done(false)

	return nil
}

func (co *ConcurrentOperationsProcessor) Close() error {
	if co.wk == nil {
		return nil
	}

	co.wk.Done(true)

	return <-co.donechan
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

	return opr, nil
}

func (co *ConcurrentOperationsProcessor) work(_ uint, j interface{}) error {
	if j == nil {
		return nil
	}

	var op state.Processor
	if sp, ok := j.(state.Processor); !ok {
		return state.IgnoreOperationProcessingError.Errorf("not state.StateProcessor, %T", j)
	} else {
		op = sp
	}

	if found, err := co.localstate.Storage().HasOperationFact(op.(operation.Operation).Fact().Hash()); err != nil {
		return err
	} else if found {
		return state.IgnoreOperationProcessingError.Errorf("already known")
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
