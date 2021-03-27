package localtime

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"
)

type testCallbackTimer struct {
	suite.Suite
}

func (t *testCallbackTimer) TestNew() {
	_, err := NewCallbackTimer(
		TimerID("good timer"),
		func(int) (bool, error) {
			return true, nil
		},
		time.Millisecond*10,
	)
	t.NoError(err)
}

func (t *testCallbackTimer) TestStart() {
	var ticked int64
	ct, err := NewCallbackTimer(
		TimerID("good timer"),
		func(int) (bool, error) {
			atomic.AddInt64(&ticked, 1)

			return true, nil
		},
		time.Millisecond*10,
	)
	t.NoError(err)

	t.NoError(ct.Start())
	t.True(xerrors.Is(ct.Start(), util.DaemonAlreadyStartedError))

	<-time.After(time.Millisecond * 50)

	ct.Stop()

	t.True(atomic.LoadInt64(&ticked) > 3)
}

func (t *testCallbackTimer) TestStop() {
	var ticked int64
	ct, err := NewCallbackTimer(
		TimerID("good timer"),
		func(int) (bool, error) {
			atomic.AddInt64(&ticked, 1)

			return true, nil
		},
		time.Millisecond*10,
	)
	t.NoError(err)

	t.NoError(ct.Start())
	t.True(xerrors.Is(ct.Start(), util.DaemonAlreadyStartedError))

	<-time.After(time.Millisecond * 40)
	ct.Stop()
	tickedStopped := atomic.LoadInt64(&ticked)

	<-time.After(time.Millisecond * 30)
	t.True(tickedStopped >= 3)
	t.True(tickedStopped <= atomic.LoadInt64(&ticked))
	t.False(ct.IsStarted())
}

func (t *testCallbackTimer) TestStoppedByCallback() {
	var ticked int64
	ct, err := NewCallbackTimer(
		TimerID("good timer"),
		func(i int) (bool, error) {
			if i == 2 {
				return false, nil // stop after calling 2 times
			}

			atomic.AddInt64(&ticked, 1)

			return true, nil
		},
		time.Millisecond*10,
	)
	ct.SetInterval(
		func(int) time.Duration {
			return time.Millisecond * 10
		},
	)
	t.NoError(err)

	t.NoError(ct.Start())
	t.True(xerrors.Is(ct.Start(), util.DaemonAlreadyStartedError))

	<-time.After(time.Millisecond * 100)
	t.True(atomic.LoadInt64(&ticked) < 4)
}

func (t *testCallbackTimer) TestStoppedByError() {
	var ticked int64
	ct, err := NewCallbackTimer(
		TimerID("good timer"),
		func(i int) (bool, error) {
			if i == 2 {
				return true, xerrors.Errorf("idontknow")
			}

			atomic.AddInt64(&ticked, 1)

			return true, nil
		},
		time.Millisecond*20,
	)
	t.NoError(err)

	t.NoError(ct.Start())
	t.True(xerrors.Is(ct.Start(), util.DaemonAlreadyStartedError))

	<-time.After(time.Millisecond * 100)
	t.True(atomic.LoadInt64(&ticked) < 4)
}

func (t *testCallbackTimer) TestIntervalFunc() {
	var ticked int64
	ct, err := NewCallbackTimer(
		TimerID("good timer"),
		func(int) (bool, error) {
			atomic.AddInt64(&ticked, 1)

			return true, nil
		},
		time.Millisecond*10,
	)

	ct.SetInterval(
		func(int) time.Duration {
			return time.Millisecond * 10
		},
	)
	t.NoError(err)

	t.NoError(ct.Start())
	t.True(xerrors.Is(ct.Start(), util.DaemonAlreadyStartedError))

	<-time.After(time.Millisecond * 60)

	ct.Stop()

	t.True(atomic.LoadInt64(&ticked) > 3)
}

func (t *testCallbackTimer) TestIntervalFuncNarrowInterval() {
	var ticked int64
	ct, err := NewCallbackTimer(
		TimerID("good timer"),
		func(int) (bool, error) {
			atomic.AddInt64(&ticked, 1)

			return true, nil
		},
		time.Millisecond*10,
	)
	ct.SetInterval(
		func(int) time.Duration {
			if atomic.LoadInt64(&ticked) > 0 { // return 0 after calling 2 times
				return 0
			}

			return time.Millisecond * 10
		},
	)
	t.NoError(err)

	t.NoError(ct.Start())
	t.True(xerrors.Is(ct.Start(), util.DaemonAlreadyStartedError))

	<-time.After(time.Millisecond * 50)

	_ = ct.Stop()

	t.True(atomic.LoadInt64(&ticked) < 4)
}

func (t *testCallbackTimer) TestLongInterval() {
	ct, err := NewCallbackTimer(
		TimerID("long-interval timer"),
		func(int) (bool, error) {
			return true, nil
		},
		time.Second*30,
	)
	t.NoError(err)
	t.NoError(ct.Start())

	<-time.After(time.Millisecond * 100)
	t.Error(xerrors.Errorf("stopping too long waited"))
	t.NoError(ct.Stop())
}

func (t *testCallbackTimer) TestRestartAfterStop() {
	var ticked int64
	ct, err := NewCallbackTimer(
		TimerID("restart timer"),
		func(int) (bool, error) {
			atomic.AddInt64(&ticked, 1)

			return true, nil
		},
		time.Millisecond*10,
	)
	ct.SetInterval(
		func(i int) time.Duration {
			if i > 2 { // stop after calling 2 times
				return 0
			}

			return time.Millisecond * 10
		},
	)
	t.NoError(err)

	t.NoError(ct.Start())

	<-time.After(time.Millisecond * 100)
	t.True(atomic.LoadInt64(&ticked) < 4)
	t.False(ct.IsStarted())

	t.NoError(ct.Restart())

	<-time.After(time.Millisecond * 100)
	t.True(atomic.LoadInt64(&ticked) > 4)
	t.True(atomic.LoadInt64(&ticked) < 8)
	t.False(ct.IsStarted())
}

func (t *testCallbackTimer) TestReset() {
	var ticked int64
	ct, err := NewCallbackTimer(
		TimerID("restart timer"),
		func(int) (bool, error) {
			atomic.AddInt64(&ticked, 1)

			return true, nil
		},
		time.Millisecond*10,
	)
	ct.SetInterval(
		func(i int) time.Duration {
			return time.Millisecond * 30
		},
	)
	t.NoError(err)

	t.NoError(ct.Start())

	<-time.After(time.Millisecond * 100)
	t.True(atomic.LoadInt64(&ticked) < 4)

	t.NoError(ct.Reset())

	<-time.After(time.Millisecond * 100)
	t.True(atomic.LoadInt64(&ticked) > 4)
	t.True(atomic.LoadInt64(&ticked) < 8)

	ct.Stop()
}

func TestCallbackTimer(t *testing.T) {
	defer goleak.VerifyNone(t)

	suite.Run(t, new(testCallbackTimer))
}
