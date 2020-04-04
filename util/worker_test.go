package util

import (
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type testWorker struct {
	suite.Suite
}

func (t *testWorker) TestRun() {
	wk := NewWorker("test-worker", 1)
	defer wk.Done()

	wk.Run(func(_ uint, job interface{}) error {
		return fmt.Errorf("%d", job)
	})

	numJob := 3

	var wg sync.WaitGroup
	wg.Add(numJob)

	var jobs []int
	for i := 0; i < numJob; i++ {
		go func(i int) {
			defer wg.Done()

			wk.NewJob(i)
		}(i)

		jobs = append(jobs, i)
	}
	wg.Wait()

	t.Equal(numJob, int(wk.Jobs()))

	var errs []int
	for err := range wk.Errors() {
		var i int
		_, err := fmt.Sscanf(err.Error(), "%d", &i)
		t.NoError(err)

		errs = append(errs, i)
		if len(errs) == numJob {
			break
		}
	}

	sort.Ints(errs)

	t.Equal(jobs, errs)
}

func (t *testWorker) TestMultipleCallbacks() {
	wk := NewWorker("test-worker", 1)
	defer wk.Done()

	numWorkers := 3

	var workers []int
	for callbackID := 0; callbackID < numWorkers; callbackID++ {
		cb := callbackID
		workers = append(workers, cb)

		wk.Run(func(_ uint, _ interface{}) error {
			return fmt.Errorf("%d", cb)
		})
	}

	for i := 0; i < numWorkers; i++ {
		wk.NewJob(i)
	}

	var called []int
	for err := range wk.Errors() {
		var i int
		_, err := fmt.Sscanf(err.Error(), "%d", &i)
		t.NoError(err)

		called = append(called, i)
		if len(called) == numWorkers {
			break
		}
	}

	t.True(wk.IsFinished())
	sort.Ints(called)

	t.Equal(workers, called)
}

func (t *testWorker) TestDoneBeforeRun() {
	wk := NewWorker("test-worker", 1)
	defer wk.Done()
}

func (t *testWorker) TestStopBeforeFinish() {
	wk := NewWorker("test-worker", 1)

	wk.Run(func(_ uint, job interface{}) error {
		<-time.After(time.Second * 100)
		return fmt.Errorf("%d", job)
	})

	numJob := 3

	var wg sync.WaitGroup
	wg.Add(numJob)

	var jobs []int
	for i := 0; i < numJob; i++ {
		go func(i int) {
			defer wg.Done()

			wk.NewJob(i)
		}(i)

		jobs = append(jobs, i)
	}
	wg.Wait()

	t.Equal(numJob, int(wk.Jobs()))

	wk.Done()
}

func TestWorker(t *testing.T) {
	suite.Run(t, new(testWorker))
}
