package util

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
	"golang.org/x/xerrors"
)

type testParallelWorker struct {
	suite.Suite
}

func (t *testParallelWorker) TestRun() {
	wk := NewParallelWorker("test-worker", 1)
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

func (t *testParallelWorker) TestMultipleCallbacks() {
	wk := NewParallelWorker("test-worker", 1)
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

func (t *testParallelWorker) TestDoneBeforeRun() {
	wk := NewParallelWorker("test-worker", 1)
	defer wk.Done()
}

func (t *testParallelWorker) TestStopBeforeFinish() {
	wk := NewParallelWorker("test-worker", 1)

	longrunningChan := make(chan struct{})
	wk.Run(func(_ uint, job interface{}) error {
		<-longrunningChan
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
	for i := 0; i < numJob; i++ {
		longrunningChan <- struct{}{}
	}

	var count int
	for _ = range wk.Errors() {
		count++
		if count == numJob {
			break
		}
	}
}

func TestParallelWorker(t *testing.T) {
	defer goleak.VerifyNone(t)

	suite.Run(t, new(testParallelWorker))
}

type testDistributeWorker struct {
	suite.Suite
}

func (t *testDistributeWorker) TestWithoutErrchan() {
	var l uint = 10
	returnChan := make(chan uint, 200000)

	wk := NewDistributeWorker(l, nil)

	go func() {
		for i := 0; i < int(l)*10; i++ {
			if !wk.NewJob(i) {
				break
			}
		}
		wk.Done(true)
	}()

	wk.Run(
		func(i uint, j interface{}) error {
			if j == nil {
				return nil
			}

			<-time.After(time.Millisecond * 10)
			returnChan <- i

			if j.(int)%3 == 0 {
				return xerrors.Errorf("error%d", j)
			}

			return nil
		},
	)

	close(returnChan)

	e := make([]uint, int(l))
	for i := uint(0); i < l; i++ {
		e[i] = i
	}
	r := make([]uint, int(l))
	for i := range returnChan {
		r[i] = uint(i)
	}

	t.Equal(e, r)
}

func (t *testDistributeWorker) TestWithErrchan() {
	var l uint = 10

	errchan := make(chan error)
	wk := NewDistributeWorker(l, errchan)

	var count int64
	go func() {
		defer wk.Done(true)

		for i := 0; i < int(l)*10; i++ {
			atomic.AddInt64(&count, 1)
			if !wk.NewJob(i) {
				break
			}
		}
	}()

	go func() {
		wk.Run(
			func(i uint, j interface{}) error {
				if j == nil {
					return nil
				}

				<-time.After(time.Millisecond * 10)

				if j.(int)%3 == 0 {
					return xerrors.Errorf("error:%d", j)
				}

				return nil
			},
		)

		close(errchan)
	}()

	var rerrs []string
	for err := range errchan {
		if err == nil {
			continue
		}

		n := strings.Split(err.Error(), ":")
		rerrs = append(rerrs, n[1])
	}
	var eerrs []string
	for i := int64(0); i < atomic.LoadInt64(&count); i++ {
		if i%3 != 0 {
			continue
		}
		eerrs = append(eerrs, fmt.Sprintf("%v", i))
	}

	sort.Strings(rerrs)
	sort.Strings(eerrs)

	t.Equal(eerrs, rerrs)
}

func (t *testDistributeWorker) TestWithErrchanStopFirst() {
	var l uint = 10

	errchan := make(chan error)
	wk := NewDistributeWorker(l, errchan)

	go func() {
		defer wk.Done(true)

		for i := 0; i < int(l)*10; i++ {
			if !wk.NewJob(i) {
				break
			}
		}
	}()

	go func() {
		wk.Run(
			func(i uint, j interface{}) error {
				if j == nil {
					return nil
				}

				if i != 0 {
					<-time.After(time.Millisecond * 100)
				}

				if j.(int)%3 == 0 {
					return xerrors.Errorf("error:%d", j)
				}

				return nil
			},
		)

		close(errchan)
	}()

	var found bool

end:
	for {
		select {
		case err := <-errchan:
			if err == nil {
				continue
			}
			found = true
			break end
		}
	}
	wk.Done(false)

	t.True(found)
}

func (t *testDistributeWorker) TestWithRunFirst() {
	var l uint = 10

	errchan := make(chan error)
	wk := NewDistributeWorker(l, errchan)

	go func() {
		defer wk.Done(true)

		for i := 0; i < int(l)*10; i++ {
			if !wk.NewJob(i) {
				break
			}
		}
	}()

	done := make(chan struct{})
	go func() {
		for _ = range errchan {
		}
		done <- struct{}{}
	}()

	wk.Run(
		func(i uint, j interface{}) error {
			if j == nil {
				return nil
			}

			<-time.After(time.Millisecond * 10)

			return nil
		},
	)

	close(errchan)

	select {
	case <-time.After(time.Second * 1):
		t.NoError(xerrors.Errorf("timeout to wait"))
	case <-done:
		//
	}
}

func (t *testDistributeWorker) TestOneCallback() {
	var l uint = 1

	errchan := make(chan error)
	wk := NewDistributeWorker(l, errchan)

	go func() {
		defer wk.Done(true)

		for i := 0; i < int(l)*10; i++ {
			if !wk.NewJob(i) {
				break
			}
		}
	}()

	go func() {
		wk.Run(
			func(i uint, j interface{}) error {
				if j == nil {
					return nil
				}

				<-time.After(time.Millisecond * 10)

				if j.(int)%3 == 0 {
					return xerrors.Errorf("error:%d", j)
				}

				return nil
			},
		)

		close(errchan)
	}()

	var found bool

end:
	for {
		select {
		case err := <-errchan:
			if err == nil {
				continue
			}

			found = true
			break end
		}
	}
	wk.Done(false)

	t.True(found)
}

func (t *testDistributeWorker) TestDoneBeforeRun() {
	{ // with errchan
		errchan := make(chan error)
		wk := NewDistributeWorker(10, errchan)
		wk.Done(true)
	}

	{ // with errchan + done false
		errchan := make(chan error)
		wk := NewDistributeWorker(10, errchan)
		wk.Done(false)
	}

	{ // without errchan
		wk := NewDistributeWorker(10, nil)
		wk.Done(true)
	}

	{ // without errchan + done false
		wk := NewDistributeWorker(10, nil)
		wk.Done(false)
	}
}

func TestDistributeWorker(t *testing.T) {
	suite.Run(t, new(testDistributeWorker))
}
