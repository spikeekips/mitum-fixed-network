package isaac

import (
	"context"
	"sync"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
)

var maxConcurrentOperations int = 500

type OperationProcessor interface {
	New(*Statepool) OperationProcessor
	Process(state.StateProcessor) error
}

type defaultOperationProcessor struct {
	pool *Statepool
}

func (opp defaultOperationProcessor) New(pool *Statepool) OperationProcessor {
	return &defaultOperationProcessor{
		pool: pool,
	}
}

func (opp defaultOperationProcessor) Process(op state.StateProcessor) error {
	return op.Process(opp.pool.Get, opp.pool.Set)
}

type ConcurrentOperationsProcessor struct {
	size       uint
	pool       *Statepool
	wk         *util.DistributeWorker
	donechan   chan error
	oprLock    sync.RWMutex
	oppHintSet *hint.Hintmap
	oprs       map[hint.Hint]OperationProcessor
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
		err := co.wk.Run(
			func(i uint, j interface{}) error {
				if j == nil {
					return nil
				} else if op, ok := j.(state.StateProcessor); !ok {
					return xerrors.Errorf("not state.StateProcessor, %T", j)
				} else if opr, err := co.opr(op); err != nil {
					return err
				} else {
					return opr.Process(op)
				}
			},
		)
		if err != nil {
			errchan <- err
		}

		close(errchan)
	}()

	go func() {
		for err := range errchan {
			if err == nil {
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

func (co *ConcurrentOperationsProcessor) Process(po operation.Operation) error {
	if co.wk == nil {
		return xerrors.Errorf("not started")
	}

	if !co.wk.NewJob(po) {
		return xerrors.Errorf("already closed")
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

func (co *ConcurrentOperationsProcessor) opr(op state.StateProcessor) (OperationProcessor, error) {
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
