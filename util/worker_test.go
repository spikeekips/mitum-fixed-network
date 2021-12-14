package util

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
	"golang.org/x/sync/semaphore"
)

type testParallelWorker struct {
	suite.Suite
}

func (t *testParallelWorker) TestRun() {
	wk := NewParallelWorker("test-worker", 1)
	defer wk.Done()

	wk.Run(func(_ uint, job interface{}) error {
		return errors.Errorf("%d", job)
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
			return errors.Errorf("%d", cb)
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
		return errors.Errorf("%d", job)
	})

	numJob := 3

	var wg sync.WaitGroup
	wg.Add(numJob)

	for i := 0; i < numJob; i++ {
		go func(i int) {
			defer wg.Done()

			wk.NewJob(i)
		}(i)
	}
	wg.Wait()

	t.Equal(numJob, int(wk.Jobs()))

	wk.Done()
	for i := 0; i < numJob; i++ {
		longrunningChan <- struct{}{}
	}

	var count int
	for range wk.Errors() {
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
	var l uint64 = 10
	returnch := make(chan uint64, l)

	callback := func(j interface{}) ContextWorkerCallback {
		return func(ctx context.Context, i uint64) error {
			if j == nil {
				return nil
			}

			select {
			case <-time.After(time.Millisecond * 10):
				returnch <- i

				if j.(uint64)%3 == 0 {
					return errors.Errorf("error%d", j)
				}

				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	wk := NewDistributeWorker(context.Background(), 5, nil)
	defer wk.Close()

	go func() {
		for i := uint64(0); i < l; i++ {
			if err := wk.NewJob(callback(i)); err != nil {
				break
			}
		}

		wk.Done()
	}()

	t.NoError(wk.Wait())

	close(returnch)

	e := make([]uint64, l)
	for i := uint64(0); i < l; i++ {
		e[i] = i
	}
	r := make([]uint64, l)
	for i := range returnch {
		r[i] = i
	}

	t.Equal(e, r)
}

func (t *testDistributeWorker) TestWithErrchan() {
	var l uint64 = 10

	errch := make(chan error)
	wk := NewDistributeWorker(context.Background(), 5, errch)
	defer wk.Close()

	callback := func(j interface{}) ContextWorkerCallback {
		return func(ctx context.Context, i uint64) error {
			if j == nil {
				return nil
			}

			select {
			case <-time.After(time.Millisecond * 10):
				if j.(uint64)%3 == 0 {
					return errors.Errorf("error:%d", j)
				}

				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	donech := make(chan struct{})

	var count int64
	go func() {
		var rerrs []string
		for err := range errch {
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

		donech <- struct{}{}
	}()

	go func() {
		for i := uint64(0); i < l; i++ {
			atomic.AddInt64(&count, 1)
			if err := wk.NewJob(callback(i)); err != nil {
				break
			}
		}

		wk.Done()
	}()

	t.NoError(wk.Wait())

	close(errch)

	<-donech
}

func (t *testDistributeWorker) TestWithErrchanCancel() {
	var l uint64 = 10

	errch := make(chan error)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wk := NewDistributeWorker(ctx, 5, errch)
	defer wk.Close()

	var called uint64
	callback := func(j interface{}) ContextWorkerCallback {
		return func(ctx context.Context, i uint64) error {
			if err := ctx.Err(); err != nil {
				return err
			}

			if j == nil {
				return nil
			}

			if i != 0 {
				<-time.After(time.Millisecond * 500)
			}

			atomic.AddUint64(&called, 1)

			if j.(uint64)%3 == 0 {
				return errors.Errorf("error:%d", j)
			}

			return nil
		}
	}

	donech := make(chan struct{})

	go func() {
		var found bool
		for err := range errch {
			if err == nil {
				continue
			} else if found {
				continue
			}

			found = true
			cancel()
		}

		t.True(found)

		donech <- struct{}{}
	}()

	go func() {
		for i := uint64(0); i < l; i++ {
			_ = wk.NewJob(callback(i))
		}

		wk.Done()
	}()

	err := wk.Wait()
	t.True(errors.Is(err, context.Canceled))

	close(errch)

	<-donech
	t.True(l > atomic.LoadUint64(&called))
}

func (t *testDistributeWorker) TestCancel() {
	var l uint64 = 10

	wk := NewDistributeWorker(context.Background(), 5, nil)
	defer wk.Close()

	var called uint64
	callback := func(j interface{}) ContextWorkerCallback {
		return func(_ context.Context, i uint64) error {
			atomic.AddUint64(&called, 1)

			<-time.After(time.Millisecond * 900)

			return nil
		}
	}

	go func() {
		for i := uint64(0); i < l; i++ {
			_ = wk.NewJob(callback(i))

			if i == 3 {
				go wk.Cancel()
			}
		}

		wk.Done()
	}()

	err := wk.Wait()
	t.NotNil(err)
	t.True(errors.Is(err, context.Canceled))

	t.True(atomic.LoadUint64(&called) < l)
}

func (t *testDistributeWorker) TestCancelBeforeRun() {
	var l uint64 = 10

	wk := NewDistributeWorker(context.Background(), 5, nil)
	defer wk.Close()

	var called uint64
	callback := func(j interface{}) ContextWorkerCallback {
		return func(context.Context, uint64) error {
			atomic.AddUint64(&called, 1)

			<-time.After(time.Millisecond * 900)

			return nil
		}
	}

	go func() {
		for i := uint64(0); i < l; i++ {
			_ = wk.NewJob(callback(i))
		}

		wk.Done()
	}()

	wk.Cancel()

	err := wk.Wait()
	t.NotNil(err)
	t.True(errors.Is(err, context.Canceled))

	t.True(atomic.LoadUint64(&called) < l)
}

func (t *testDistributeWorker) TestLazyCancel() {
	var l uint64 = 10

	var called, canceled uint64
	callback := func(j interface{}) ContextWorkerCallback {
		return func(ctx context.Context, jobid uint64) error {
			if jobid < 3 {
				atomic.AddUint64(&called, 1)

				return nil
			}

			select {
			case <-time.After(time.Second * 900):
			case <-ctx.Done():
				atomic.AddUint64(&canceled, 1)
			}

			return nil
		}
	}

	wk := NewDistributeWorker(context.Background(), int64(l), nil)
	defer wk.Close()

	go func() {
		for i := uint64(0); i < l; i++ {
			_ = wk.NewJob(callback(i))
		}

		wk.LazyCancel(time.Millisecond * 300)
	}()

	err := wk.Wait()
	t.NotNil(err)
	t.True(errors.Is(err, context.Canceled))

	t.True(atomic.LoadUint64(&canceled) < l)

	<-time.After(time.Second)
	t.True(atomic.LoadUint64(&canceled) < l)
}

func TestDistributeWorker(t *testing.T) {
	defer goleak.VerifyNone(t)

	sem := semaphore.NewWeighted(100)
	for i := 0; i < 1000; i++ {
		_ = sem.Acquire(context.Background(), 1)

		go func() {
			defer sem.Release(1)

			suite.Run(t, new(testDistributeWorker))
		}()
	}

	_ = sem.Acquire(context.Background(), 100)
}

type testErrgroupWorker struct {
	suite.Suite
}

func (t *testErrgroupWorker) TestNoError() {
	var l uint64 = 10
	returnch := make(chan uint64, l)

	callback := func(j interface{}) ContextWorkerCallback {
		return func(ctx context.Context, i uint64) error {
			if j == nil {
				return nil
			}

			select {
			case <-time.After(time.Millisecond * 10):
				returnch <- i
			case <-ctx.Done():
				return ctx.Err()
			}

			return nil
		}
	}

	wk := NewErrgroupWorker(context.Background(), int64(l))
	defer wk.Close()

	go func() {
		for i := uint64(0); i < l; i++ {
			if err := wk.NewJob(callback(i)); err != nil {
				break
			}
		}

		wk.Done()
	}()

	t.NoError(wk.Wait())

	close(returnch)

	e := make([]uint64, l)
	for i := uint64(0); i < l; i++ {
		e[i] = i
	}
	r := make([]uint64, l)
	for i := range returnch {
		r[i] = i
	}

	t.Equal(e, r)
}

func (t *testErrgroupWorker) TestError() {
	var l uint64 = 10

	var called uint64
	callback := func(j interface{}) ContextWorkerCallback {
		return func(ctx context.Context, i uint64) error {
			if j == nil {
				return nil
			}

			if i := j.(uint64); i == 3 {
				return errors.Errorf("error:%d", j)
			}

			select {
			case <-time.After(time.Second):
				atomic.AddUint64(&called, 1)
			case <-ctx.Done():
				return ctx.Err()
			}

			return nil
		}
	}

	wk := NewErrgroupWorker(context.Background(), int64(l))
	defer wk.Close()

	go func() {
		for i := uint64(0); i < l; i++ {
			if err := wk.NewJob(callback(i)); err != nil {
				break
			}
		}

		wk.Done()
	}()

	err := wk.Wait()
	t.NotNil(err)
	t.Contains(err.Error(), "error:3")

	c := atomic.LoadUint64(&called)
	t.True(c < 1, "called=%d", c)
}

func (t *testErrgroupWorker) TestDeadlineError() {
	var l uint64 = 10

	var called uint64
	callback := func(j interface{}) ContextWorkerCallback {
		return func(ctx context.Context, i uint64) error {
			if j == nil {
				return nil
			}

			select {
			case <-time.After(time.Millisecond * 900):
				atomic.AddUint64(&called, 1)
			case <-ctx.Done():
				return ctx.Err()
			}

			return nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()

	wk := NewErrgroupWorker(ctx, int64(l))
	defer wk.Close()

	go func() {
		for i := uint64(0); i < l; i++ {
			if err := wk.NewJob(callback(i)); err != nil {
				break
			}
		}

		wk.Done()
	}()

	err := wk.Wait()
	t.True(errors.Is(err, context.DeadlineExceeded))

	c := atomic.LoadUint64(&called)
	t.T().Logf("called: %d", c)
	t.True(c < 1)
}

func (t *testErrgroupWorker) TestCancel() {
	var l uint64 = 10

	callback := func(j interface{}) ContextWorkerCallback {
		return func(context.Context, uint64) error {
			<-time.After(time.Millisecond * 900)

			return nil
		}
	}

	wk := NewErrgroupWorker(context.Background(), int64(l))
	defer wk.Close()

	go func() {
		for i := uint64(0); i < l; i++ {
			_ = wk.NewJob(callback(i))

			if i == 3 {
				go wk.Cancel()
			}
		}

		wk.Done()
	}()

	t.NoError(wk.Wait())
}

func (t *testErrgroupWorker) TestCancelBeforeRun() {
	var l uint64 = 10

	var called uint64
	callback := func(j interface{}) ContextWorkerCallback {
		return func(context.Context, uint64) error {
			atomic.AddUint64(&called, 1)
			<-time.After(time.Millisecond * 900)

			return nil
		}
	}

	wk := NewErrgroupWorker(context.Background(), int64(l)-5)
	defer wk.Close()

	go func() {
		for i := uint64(0); i < l; i++ {
			_ = wk.NewJob(callback(i))
		}

		wk.Done()
	}()

	wk.Cancel()

	t.NoError(wk.Wait())

	c := atomic.LoadUint64(&called)
	t.True(c < l, "called=%d, total=%d", c, l)
}

func (t *testErrgroupWorker) TestLazyClose() {
	var l uint64 = 10

	var called, canceled uint64
	callback := func(j interface{}) ContextWorkerCallback {
		return func(ctx context.Context, jobid uint64) error {
			if jobid < 3 {
				atomic.AddUint64(&called, 1)

				return nil
			}

			select {
			case <-time.After(time.Second * 900):
			case <-ctx.Done():
				atomic.AddUint64(&canceled, 1)
			}

			return nil
		}
	}

	wk := NewErrgroupWorker(context.Background(), int64(l))
	defer wk.Close()

	go func() {
		for i := uint64(0); i < l; i++ {
			_ = wk.NewJob(callback(i))
		}

		wk.LazyCancel(time.Millisecond * 100)
	}()

	err := wk.Wait()
	t.NotNil(err)
	t.True(errors.Is(err, context.Canceled))

	t.True(atomic.LoadUint64(&canceled) < l)

	<-time.After(time.Millisecond * 500)
	t.True(atomic.LoadUint64(&canceled) < l)
}

func TestErrgroupWorker(t *testing.T) {
	defer goleak.VerifyNone(t)

	sem := semaphore.NewWeighted(100)
	for i := 0; i < 1000; i++ {
		_ = sem.Acquire(context.Background(), 1)

		go func() {
			defer sem.Release(1)

			suite.Run(t, new(testErrgroupWorker))
		}()
	}

	_ = sem.Acquire(context.Background(), 100)
}
