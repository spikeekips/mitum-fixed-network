package mitum

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testCallbackTimer struct {
	suite.Suite
}

func (t *testCallbackTimer) TestNew() {
	_, err := NewCallbackTimer(
		"good timer",
		func() (error, bool) {
			return nil, true
		},
		time.Millisecond*10,
		nil,
	)
	t.NoError(err)
}

func (t *testCallbackTimer) TestStart() {
	var ticked int64
	ct, err := NewCallbackTimer(
		"good timer",
		func() (error, bool) {
			atomic.AddInt64(&ticked, 1)

			return nil, true
		},
		time.Millisecond*10,
		nil,
	)
	t.NoError(err)

	t.NoError(ct.Start())
	t.True(xerrors.Is(ct.Start(), DaemonAlreadyStartedError))

	time.Sleep(time.Millisecond * 50)

	ct.Stop()

	t.True(atomic.LoadInt64(&ticked) > 3)
}

func (t *testCallbackTimer) TestStop() {
	var ticked int64
	ct, err := NewCallbackTimer(
		"good timer",
		func() (error, bool) {
			atomic.AddInt64(&ticked, 1)
			return nil, true
		},
		time.Millisecond*10,
		nil,
	)
	t.NoError(err)

	t.NoError(ct.Start())
	t.True(xerrors.Is(ct.Start(), DaemonAlreadyStartedError))

	time.Sleep(time.Millisecond * 30)
	ct.Stop()
	tickedStopped := atomic.LoadInt64(&ticked)

	time.Sleep(time.Millisecond * 30)
	t.Equal(tickedStopped, atomic.LoadInt64(&ticked))
}

func (t *testCallbackTimer) TestStoppedByCallback() {
	var ticked int64
	ct, err := NewCallbackTimer(
		"good timer",
		func() (error, bool) {
			atomic.AddInt64(&ticked, 1)

			return nil, atomic.LoadInt64(&ticked) < 2 // stop after calling 2 times
		},
		time.Millisecond*10,
		nil,
	)
	t.NoError(err)

	t.NoError(ct.Start())
	t.True(xerrors.Is(ct.Start(), DaemonAlreadyStartedError))

	time.Sleep(time.Millisecond * 100)
	t.True(ct.IsStopped())
	t.True(atomic.LoadInt64(&ticked) < 4)
}

func (t *testCallbackTimer) TestStoppedByError() {
	var ticked int64
	ct, err := NewCallbackTimer(
		"good timer",
		func() (error, bool) {
			atomic.AddInt64(&ticked, 1)

			if atomic.LoadInt64(&ticked) < 2 {
				return nil, true
			}

			return xerrors.Errorf("idontknow"), true
		},
		time.Millisecond*10,
		nil,
	)
	t.NoError(err)

	t.NoError(ct.Start())
	t.True(xerrors.Is(ct.Start(), DaemonAlreadyStartedError))

	time.Sleep(time.Millisecond * 100)
	t.True(ct.IsStopped())
	t.True(atomic.LoadInt64(&ticked) < 4)
}

func (t *testCallbackTimer) TestIntervalFunc() {
	var ticked int64
	ct, err := NewCallbackTimer(
		"good timer",
		func() (error, bool) {
			atomic.AddInt64(&ticked, 1)

			return nil, true
		},
		0,
		func() time.Duration {
			return time.Millisecond * 10
		},
	)
	t.NoError(err)

	t.NoError(ct.Start())
	t.True(xerrors.Is(ct.Start(), DaemonAlreadyStartedError))

	time.Sleep(time.Millisecond * 50)

	ct.Stop()

	t.True(atomic.LoadInt64(&ticked) > 3)
}

func (t *testCallbackTimer) TestIntervalFuncNarrowInterval() {
	var ticked int64
	ct, err := NewCallbackTimer(
		"good timer",
		func() (error, bool) {
			atomic.AddInt64(&ticked, 1)

			return nil, true
		},
		0,
		func() time.Duration {
			if atomic.LoadInt64(&ticked) > 0 { // return 0 after calling 2 times
				return 0
			}

			return time.Millisecond * 10
		},
	)
	t.NoError(err)

	t.NoError(ct.Start())
	t.True(xerrors.Is(ct.Start(), DaemonAlreadyStartedError))

	time.Sleep(time.Millisecond * 50)

	ct.Stop()

	t.True(atomic.LoadInt64(&ticked) < 4)
}

func (t *testCallbackTimer) TestCallbackTimerset() {
	n := 3
	var timers []*CallbackTimer

	var tickeds []*int64

	for i := 0; i < n; i++ {
		i := i
		ticked := new(int64)
		tickeds = append(tickeds, ticked)

		ct, err := NewCallbackTimer(
			fmt.Sprintf("good timer: %d", i),
			func() (error, bool) {
				atomic.AddInt64(ticked, 1)

				return nil, true
			},
			time.Millisecond*10,
			nil,
		)
		t.NoError(err)
		timers = append(timers, ct)
	}
	cts := NewCallbackTimerset(timers)
	t.NoError(cts.Start())

	time.Sleep(time.Millisecond * 50)
	t.NoError(cts.Stop())

	for _, ticked := range tickeds {
		t.True(atomic.LoadInt64(ticked) > 3)
	}
}

func TestCallbackTimer(t *testing.T) {
	suite.Run(t, new(testCallbackTimer))
}