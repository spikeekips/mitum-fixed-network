package util

import (
	"fmt"
	"sync"

	"github.com/spikeekips/mitum/util/logging"
)

type WorkerCallback func( /* job id */ uint, interface{} /* arguments */) error

type Worker struct {
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

func NewWorker(name string, bufsize uint) *Worker {
	wk := &Worker{
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

func (wk *Worker) roundrobin() {
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

func (wk *Worker) Run(callback WorkerCallback) *Worker {
	wk.Lock()
	defer wk.Unlock()

	wk.callbacks = append(wk.callbacks, callback)

	return wk
}

func (wk *Worker) nextCallback() WorkerCallback {
	wk.Lock()
	defer wk.Unlock()

	index := wk.lastCalled + 1

	if index >= len(wk.callbacks) {
		index = 0
	}

	wk.lastCalled = index

	return wk.callbacks[index]
}

func (wk *Worker) NewJob(j interface{}) {
	wk.Lock()
	wk.jobCalled++
	wk.Unlock()

	wk.Log().Debug().
		Interface("arguments", j).
		Msg("new job")

	wk.jobChan <- j
}

func (wk *Worker) Errors() <-chan error {
	return wk.errChan
}

func (wk *Worker) Jobs() uint {
	wk.RLock()
	defer wk.RUnlock()

	return wk.jobCalled
}

func (wk *Worker) FinishedJobs() int {
	wk.RLock()
	defer wk.RUnlock()

	return wk.jobFinished
}

func (wk *Worker) Done() {
	if wk.jobChan != nil {
		close(wk.jobChan)
	}

	// NOTE don't close errChan :)
}

func (wk *Worker) IsFinished() bool {
	wk.RLock()
	defer wk.RUnlock()

	return uint(wk.jobFinished) == wk.jobCalled
}
