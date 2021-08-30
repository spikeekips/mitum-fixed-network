package util

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

type (
	WorkerCallback        func( /* job id */ uint, interface{} /* arguments */) error
	ContextWorkerCallback func(context.Context, uint64 /* job id */) error
)

type contextCanceled struct{}

func (contextCanceled) Error() string {
	return "context canceled in worker"
}

type ParallelWorker struct {
	sync.RWMutex
	*logging.Logging
	jobChan     chan interface{}
	errChan     chan error
	bufsize     uint
	jobCalled   uint
	jobFinished int
	callbacks   []WorkerCallback
	lastCalled  int
}

func NewParallelWorker(name string, bufsize uint) *ParallelWorker {
	wk := &ParallelWorker{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", fmt.Sprintf("worker-%s", name))
		}),
		bufsize:    bufsize,
		jobChan:    make(chan interface{}, int(bufsize)),
		errChan:    make(chan error),
		lastCalled: -1,
	}

	go wk.roundrobin()

	return wk
}

func (wk *ParallelWorker) roundrobin() {
	var jobID uint
	for job := range wk.jobChan {
		callback := wk.nextCallback()
		go func(jobID uint, job interface{}) {
			err := callback(jobID, job)

			wk.Lock()
			wk.jobFinished++
			wk.Unlock()

			wk.errChan <- err
		}(jobID, job)
		jobID++
	}
}

func (wk *ParallelWorker) Run(callback WorkerCallback) *ParallelWorker {
	wk.Lock()
	defer wk.Unlock()

	wk.callbacks = append(wk.callbacks, callback)

	return wk
}

func (wk *ParallelWorker) nextCallback() WorkerCallback {
	wk.Lock()
	defer wk.Unlock()

	index := wk.lastCalled + 1

	if index >= len(wk.callbacks) {
		index = 0
	}

	wk.lastCalled = index

	return wk.callbacks[index]
}

func (wk *ParallelWorker) NewJob(j interface{}) {
	wk.Lock()
	wk.jobCalled++
	wk.Unlock()

	wk.Log().Debug().
		Interface("arguments", j).
		Msg("new job")

	wk.jobChan <- j
}

func (wk *ParallelWorker) Errors() <-chan error {
	return wk.errChan
}

func (wk *ParallelWorker) Jobs() uint {
	wk.RLock()
	defer wk.RUnlock()

	return wk.jobCalled
}

func (wk *ParallelWorker) FinishedJobs() int {
	wk.RLock()
	defer wk.RUnlock()

	return wk.jobFinished
}

func (wk *ParallelWorker) Done() {
	if wk.jobChan != nil {
		close(wk.jobChan)
	}
	// NOTE don't close errChan :)
}

func (wk *ParallelWorker) IsFinished() bool {
	wk.RLock()
	defer wk.RUnlock()

	return uint(wk.jobFinished) == wk.jobCalled
}

type BaseSemWorker struct {
	N          int64
	Sem        *semaphore.Weighted
	Ctx        context.Context
	Cancel     func()
	JobCount   uint64
	NewJobFunc func(context.Context, uint64, ContextWorkerCallback)
	runonce    sync.Once
	donech     chan time.Duration
}

func NewBaseSemWorker(ctx context.Context, semsize int64) *BaseSemWorker {
	closectx, cancel := context.WithCancel(ctx)

	return &BaseSemWorker{
		N:      semsize,
		Sem:    semaphore.NewWeighted(semsize),
		Ctx:    closectx,
		Cancel: cancel,
		donech: make(chan time.Duration, 2),
	}
}

func (wk *BaseSemWorker) NewJob(callback ContextWorkerCallback) error {
	if err := wk.Ctx.Err(); err != nil {
		return err
	}

	jobs := wk.JobCount

	if err := wk.Sem.Acquire(wk.Ctx, 1); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(wk.Ctx)
	go func() {
		defer wk.Sem.Release(1)
		defer cancel()

		wk.NewJobFunc(ctx, jobs, callback)
	}()
	wk.JobCount++

	return nil
}

func (wk *BaseSemWorker) Jobs() uint64 {
	return wk.JobCount
}

func (wk *BaseSemWorker) Wait() error {
	err := wk.wait()
	if err != nil {
		if errors.Is(err, contextCanceled{}) {
			return context.Canceled
		}

		return err
	}

	return nil
}

func (wk *BaseSemWorker) wait() error {
	errch := make(chan error, 1)
	wk.runonce.Do(func() {
		timeout := <-wk.donech

		donech := make(chan error, 1)
		go func() {
			switch err := wk.Sem.Acquire(context.Background(), wk.N); {
			case err != nil:
				donech <- err
			default:
				donech <- wk.Ctx.Err()
			}
		}()

		if timeout < 1 {
			err := <-donech
			errch <- err

			return
		}

		select {
		case <-time.After(timeout):
			wk.Cancel()

			errch <- contextCanceled{}
		case err := <-donech:
			errch <- err
		}
	})

	return <-errch
}

func (wk *BaseSemWorker) WaitChan() chan error {
	ch := make(chan error)
	go func() {
		ch <- wk.Wait()
	}()

	return ch
}

func (wk *BaseSemWorker) Done() {
	wk.donech <- 0
}

func (wk *BaseSemWorker) Close() {
	wk.donech <- 0

	wk.Cancel()
}

func (wk *BaseSemWorker) LazyClose(timeout time.Duration) {
	wk.donech <- timeout
}

type DistributeWorker struct {
	*BaseSemWorker
	errch chan error
}

func NewDistributeWorker(ctx context.Context, semsize int64, errch chan error) *DistributeWorker {
	base := NewBaseSemWorker(ctx, semsize)

	var errf func(error)
	if errch == nil {
		errf = func(error) {}
	} else {
		errf = func(err error) {
			if cerr := base.Ctx.Err(); cerr == nil {
				errch <- err
			}
		}
	}

	base.NewJobFunc = func(ctx context.Context, jobs uint64, callback ContextWorkerCallback) {
		errf(callback(ctx, jobs))
	}

	return &DistributeWorker{
		BaseSemWorker: base,
		errch:         errch,
	}
}

type ErrgroupWorker struct {
	*BaseSemWorker
	eg        *errgroup.Group
	egLock    sync.Mutex
	doneonece sync.Once
}

func NewErrgroupWorker(ctx context.Context, semsize int64) *ErrgroupWorker {
	base := NewBaseSemWorker(ctx, semsize)

	eg, egctx := errgroup.WithContext(base.Ctx)
	base.Ctx = egctx

	wk := &ErrgroupWorker{
		BaseSemWorker: base,
		eg:            eg,
	}

	base.NewJobFunc = func(ctx context.Context, jobs uint64, callback ContextWorkerCallback) {
		donech := make(chan struct{}, 1)
		wk.egGo(func() error {
			defer func() {
				donech <- struct{}{}
			}()

			return callback(ctx, jobs)
		})

		<-donech
	}

	return wk
}

func (wk *ErrgroupWorker) Wait() error {
	if err := wk.BaseSemWorker.wait(); err != nil {
		if !errors.Is(err, context.Canceled) {
			if errors.Is(err, contextCanceled{}) {
				return context.Canceled
			}

			return err
		}
	}

	errch := make(chan error, 1)
	wk.doneonece.Do(func() {
		wk.egLock.Lock()
		defer wk.egLock.Unlock()

		errch <- wk.eg.Wait()
	})

	return <-errch
}

func (wk *ErrgroupWorker) RunChan() chan error {
	ch := make(chan error)
	go func() {
		ch <- wk.Wait()
	}()

	return ch
}

func (wk *ErrgroupWorker) egGo(f func() error) {
	wk.egLock.Lock()
	defer wk.egLock.Unlock()

	wk.eg.Go(f)
}
