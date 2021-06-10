package util

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

type WorkerCallback func( /* job id */ uint, interface{} /* arguments */) error

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
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
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

type DistributeWorker struct {
	sync.RWMutex
	n          uint
	wg         *sync.WaitGroup
	input      chan interface{}
	closed     bool
	closedchan chan uint
	errchan    chan error
	sonce      sync.Once
	conce      sync.Once
	jobs       uint64
}

func NewDistributeWorker(n uint, errchan chan error) *DistributeWorker {
	return &DistributeWorker{
		n:          n,
		input:      make(chan interface{}),
		closedchan: make(chan uint, n),
		errchan:    errchan,
	}
}

func (wk *DistributeWorker) Run(callback WorkerCallback) error {
	var errcallback func(error)
	if wk.errchan == nil {
		errcallback = func(error) {}
	} else {
		errcallback = func(err error) {
			wk.errchan <- err
		}
	}

	if wk.wg != nil {
		return xerrors.Errorf("already ran")
	}

	wk.wg = &sync.WaitGroup{}
	wk.wg.Add(int(wk.n))

	for i := uint(0); i < wk.n; i++ {
		go func(i uint) {
			defer wk.wg.Done()

		end:
			for {
				select {
				case <-wk.closedchan:
					break end
				case j := <-wk.input:
					errcallback(callback(i, j))
				}
			}
		}(i)
	}

	wk.wg.Wait()

	return nil
}

func (wk *DistributeWorker) NewJob(i interface{}) bool {
	if func() bool {
		wk.RLock()
		defer wk.RUnlock()

		return wk.closed
	}() {
		return false
	}

	atomic.AddUint64(&wk.jobs, 1)
	wk.input <- i

	return true
}

func (wk *DistributeWorker) Jobs() uint64 {
	return atomic.LoadUint64(&wk.jobs)
}

func (wk *DistributeWorker) Done(setClose bool) {
	wk.sonce.Do(func() {
		wk.Lock()
		wk.closed = true
		wk.Unlock()

		for i := uint(0); i < wk.n; i++ {
			wk.closedchan <- i
		}
	})

	if setClose {
		wk.conce.Do(func() {
			close(wk.input)
			close(wk.closedchan)
		})
	}
}
