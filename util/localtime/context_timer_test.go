package localtime

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
	"golang.org/x/sync/semaphore"
)

type testContextTimer struct {
	suite.Suite
}

func (t *testContextTimer) TestNew() {
	_ = NewContextTimer(
		TimerID("good timer"),
		time.Millisecond*10,
		func(int) (bool, error) {
			return true, nil
		},
	)
}

func (t *testContextTimer) TestStart() {
	var ticked int64
	ct := NewContextTimer(
		TimerID("good timer"),
		time.Millisecond*10,
		func(i int) (bool, error) {
			atomic.AddInt64(&ticked, 1)

			return true, nil
		},
	)

	t.NoError(ct.Start())
	t.True(errors.Is(ct.Start(), util.DaemonAlreadyStartedError))

	<-time.After(time.Millisecond * 50)

	t.NoError(ct.Stop())

	t.True(atomic.LoadInt64(&ticked) > 3)
}

func (t *testContextTimer) TestStop() {
	var ticked int64
	ct := NewContextTimer(
		TimerID("good timer"),
		time.Millisecond*10,
		func(int) (bool, error) {
			atomic.AddInt64(&ticked, 1)

			return true, nil
		},
	)

	t.NoError(ct.Start())
	t.True(errors.Is(ct.Start(), util.DaemonAlreadyStartedError))

	<-time.After(time.Millisecond * 40)
	ct.Stop()
	tickedStopped := atomic.LoadInt64(&ticked)

	<-time.After(time.Millisecond * 30)
	t.True(tickedStopped >= 3)
	t.True(tickedStopped <= atomic.LoadInt64(&ticked))
	t.False(ct.IsStarted())
}

func (t *testContextTimer) TestStoppedByCallback() {
	var ticked int64
	ct := NewContextTimer(
		TimerID("good timer"),
		time.Millisecond*10,
		func(i int) (bool, error) {
			if i == 2 {
				return false, nil // stop after calling 2 times
			}

			atomic.AddInt64(&ticked, 1)

			return true, nil
		},
	)
	_ = ct.SetInterval(func(int) time.Duration {
		return time.Millisecond * 10
	})

	t.NoError(ct.Start())
	t.True(errors.Is(ct.Start(), util.DaemonAlreadyStartedError))

	<-time.After(time.Millisecond * 100)
	t.True(atomic.LoadInt64(&ticked) < 4)
}

func (t *testContextTimer) TestIntervalFunc() {
	var ticked int64
	ct := NewContextTimer(
		TimerID("good timer"),
		time.Millisecond*10,
		func(int) (bool, error) {
			atomic.AddInt64(&ticked, 1)

			return true, nil
		},
	)

	_ = ct.SetInterval(func(int) time.Duration {
		return time.Millisecond * 10
	})

	t.NoError(ct.Start())
	t.True(errors.Is(ct.Start(), util.DaemonAlreadyStartedError))

	<-time.After(time.Millisecond * 60)

	ct.Stop()

	t.True(atomic.LoadInt64(&ticked) > 3)
}

func (t *testContextTimer) TestIntervalFuncNarrowInterval() {
	var ticked int64
	ct := NewContextTimer(
		TimerID("good timer"),
		time.Millisecond*10,
		func(int) (bool, error) {
			atomic.AddInt64(&ticked, 1)

			return true, nil
		},
	)
	_ = ct.SetInterval(func(int) time.Duration {
		if atomic.LoadInt64(&ticked) > 0 { // return 0 after calling 2 times
			return 0
		}

		return time.Millisecond * 10
	})

	t.NoError(ct.Start())
	t.True(errors.Is(ct.Start(), util.DaemonAlreadyStartedError))

	<-time.After(time.Millisecond * 50)

	_ = ct.Stop()

	t.True(atomic.LoadInt64(&ticked) < 4)
}

func (t *testContextTimer) TestLongInterval() {
	ct := NewContextTimer(
		TimerID("long-interval timer"),
		time.Second*30,
		func(int) (bool, error) {
			return true, nil
		},
	)
	t.NoError(ct.Start())

	<-time.After(time.Millisecond * 100)
	t.NoError(ct.Stop())
}

func (t *testContextTimer) TestLongRunning() {
	sem := semaphore.NewWeighted(50)

	ctx := context.Background()

	var run uint64
	for i := 0; i < 100; i++ {
		if err := sem.Acquire(ctx, 1); err != nil {
			panic(err)
		}

		i := i
		go func() {
			defer sem.Release(1)
			defer func() {
				atomic.AddUint64(&run, 1)

				if n := atomic.LoadUint64(&run); n%20 == 0 {
					t.T().Logf("< % 3d: % 3d", i, n)
				}
			}()

			stopch := make(chan bool, 1)
			var once sync.Once

			ct := NewContextTimer(
				TimerID("long-interval timer"),
				time.Millisecond*100,
				func(int) (bool, error) {
					defer once.Do(func() {
						stopch <- true
					})

					<-time.After(time.Second * 2)
					return true, nil
				},
			)
			t.NoError(ct.Start())

			<-time.After(time.Second)
			t.NoError(ct.Stop())
			<-stopch
		}()
	}

	t.NoError(sem.Acquire(ctx, 50))
	t.T().Log("done")
}

func (t *testContextTimer) TestRestartAfterStop() {
	var ticked int64
	ct := NewContextTimer(
		TimerID("restart timer"),
		time.Millisecond*10,
		func(i int) (bool, error) {
			atomic.AddInt64(&ticked, 1)

			return true, nil
		},
	)
	_ = ct.SetInterval(func(i int) time.Duration {
		if i > 2 { // stop after calling 2 times
			return 0
		}

		return time.Millisecond * 10
	})

	t.NoError(ct.Start())

	<-time.After(time.Millisecond * 200)
	t.True(atomic.LoadInt64(&ticked) < 4)
	t.False(ct.IsStarted())

	t.NoError(ct.Start())

	<-time.After(time.Millisecond * 100)
	t.True(atomic.LoadInt64(&ticked) > 4)
	t.True(atomic.LoadInt64(&ticked) < 8)
	t.False(ct.IsStarted())
}

func (t *testContextTimer) TestReset() {
	var ticked int64
	ct := NewContextTimer(
		TimerID("restart timer"),
		time.Millisecond*10,
		func(i int) (bool, error) {
			atomic.AddInt64(&ticked, 1)

			return true, nil
		},
	)
	_ = ct.SetInterval(func(i int) time.Duration {
		return time.Millisecond * 30
	})

	t.NoError(ct.Start())
	defer ct.Stop()

	<-time.After(time.Millisecond * 100)
	t.True(atomic.LoadInt64(&ticked) < 4)

	t.NoError(ct.Reset())

	<-time.After(time.Millisecond * 100)
	t.True(atomic.LoadInt64(&ticked) > 4)
	t.True(atomic.LoadInt64(&ticked) < 8)
}

func TestContextTimer(t *testing.T) {
	defer goleak.VerifyNone(t)

	suite.Run(t, new(testContextTimer))
}
