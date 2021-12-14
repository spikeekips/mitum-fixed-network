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

var baseSemWorkerPool = sync.Pool{
	New: func() interface{} {
		return new(BaseSemWorker)
	},
}

var baseSemWorkerPoolPut = func(wk *BaseSemWorker) {
	wk.N = 0
	wk.Sem = nil
	wk.Ctx = nil
	wk.Cancel = nil
	wk.JobCount = 0
	wk.NewJobFunc = nil
	wk.donech = nil

	baseSemWorkerPool.Put(wk)
}

var distributeWorkerPool = sync.Pool{
	New: func() interface{} {
		return new(DistributeWorker)
	},
}

var distributeWorkerPoolPut = func(wk *DistributeWorker) {
	wk.BaseSemWorker = nil
	wk.errch = nil

	distributeWorkerPool.Put(wk)
}

var errgroupWorkerPool = sync.Pool{
	New: func() interface{} {
		return new(ErrgroupWorker)
	},
}

var errgroupWorkerPoolPut = func(wk *ErrgroupWorker) {
	wk.BaseSemWorker = nil
	wk.eg = nil

	errgroupWorkerPool.Put(wk)
}

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
	wk := baseSemWorkerPool.Get().(*BaseSemWorker)
	closectx, cancel := context.WithCancel(ctx)

	wk.N = semsize
	wk.Sem = semaphore.NewWeighted(semsize)
	wk.Ctx = closectx
	wk.Cancel = cancel
	wk.JobCount = 0
	wk.runonce = sync.Once{}
	wk.donech = make(chan time.Duration, 2)

	return wk
}

func (wk *BaseSemWorker) NewJob(callback ContextWorkerCallback) error {
	if err := wk.Ctx.Err(); err != nil {
		return err
	}

	sem := wk.Sem
	newjob := wk.NewJobFunc
	jobs := wk.JobCount

	if err := wk.Sem.Acquire(wk.Ctx, 1); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(wk.Ctx)
	go func() {
		defer sem.Release(1)
		defer cancel()

		newjob(ctx, jobs, callback)
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
	n := wk.N
	sem := wk.Sem
	ctx := wk.Ctx
	cancel := wk.Cancel

	errch := make(chan error, 1)

	wk.runonce.Do(func() {
		timeout := <-wk.donech

		donech := make(chan error, 1)
		go func() {
			switch err := sem.Acquire(context.Background(), n); { // nolint:contextcheck
			case err != nil:
				donech <- err
			default:
				donech <- ctx.Err()
			}
		}()

		if timeout < 1 {
			errch <- <-donech

			return
		}

		select {
		case <-time.After(timeout):
			cancel()

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
	defer baseSemWorkerPoolPut(wk)

	wk.donech <- 0

	wk.Cancel()
}

func (wk *BaseSemWorker) LazyCancel(timeout time.Duration) {
	wk.donech <- timeout
}

type DistributeWorker struct {
	*BaseSemWorker
	errch chan error
}

func NewDistributeWorker(ctx context.Context, semsize int64, errch chan error) *DistributeWorker {
	wk := distributeWorkerPool.Get().(*DistributeWorker)

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

	wk.BaseSemWorker = base
	wk.errch = errch

	return wk
}

func (wk *DistributeWorker) Close() {
	defer distributeWorkerPoolPut(wk)

	wk.BaseSemWorker.Close()
}

type ErrgroupWorker struct {
	*BaseSemWorker
	eg        *errgroup.Group
	doneonece sync.Once
}

func NewErrgroupWorker(ctx context.Context, semsize int64) *ErrgroupWorker {
	wk := errgroupWorkerPool.Get().(*ErrgroupWorker)

	base := NewBaseSemWorker(ctx, semsize)

	eg, egctx := errgroup.WithContext(base.Ctx)
	base.Ctx = egctx

	base.NewJobFunc = func(ctx context.Context, jobs uint64, callback ContextWorkerCallback) {
		donech := make(chan struct{}, 1)
		eg.Go(func() error {
			defer func() {
				donech <- struct{}{}
			}()

			return callback(ctx, jobs)
		})

		<-donech
	}

	wk.BaseSemWorker = base
	wk.eg = eg
	wk.doneonece = sync.Once{}

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
		errch <- wk.eg.Wait()
	})

	return <-errch
}

func (wk *ErrgroupWorker) Close() {
	defer errgroupWorkerPoolPut(wk)

	wk.BaseSemWorker.Close()
}

func (wk *ErrgroupWorker) RunChan() chan error {
	ch := make(chan error)
	go func() {
		ch <- wk.Wait()
	}()

	return ch
}
