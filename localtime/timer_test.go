package localtime

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"
)

type testCallbackTimer struct {
	suite.Suite
}

func (t *testCallbackTimer) TestNew() {
	_, err := NewCallbackTimer(
		"good timer",
		func() (bool, error) {
			return true, nil
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
		func() (bool, error) {
			atomic.AddInt64(&ticked, 1)

			return true, nil
		},
		time.Millisecond*10,
		nil,
	)
	t.NoError(err)

	t.NoError(ct.Start())
	t.True(xerrors.Is(ct.Start(), util.DaemonAlreadyStartedError))

	time.Sleep(time.Millisecond * 50)

	ct.Stop()

	t.True(atomic.LoadInt64(&ticked) > 3)
}

func (t *testCallbackTimer) TestStop() {
	var ticked int64
	ct, err := NewCallbackTimer(
		"good timer",
		func() (bool, error) {
			atomic.AddInt64(&ticked, 1)
			return true, nil
		},
		time.Millisecond*10,
		nil,
	)
	t.NoError(err)

	t.NoError(ct.Start())
	t.True(xerrors.Is(ct.Start(), util.DaemonAlreadyStartedError))

	time.Sleep(time.Millisecond * 40)
	ct.Stop()
	tickedStopped := atomic.LoadInt64(&ticked)

	time.Sleep(time.Millisecond * 30)
	t.True(tickedStopped >= 3)
	t.True(tickedStopped <= atomic.LoadInt64(&ticked))
}

func (t *testCallbackTimer) TestStoppedByCallback() {
	var ticked int64
	ct, err := NewCallbackTimer(
		"good timer",
		func() (bool, error) {
			atomic.AddInt64(&ticked, 1)

			return atomic.LoadInt64(&ticked) < 2, nil // stop after calling 2 times
		},
		time.Millisecond*10,
		nil,
	)
	t.NoError(err)

	t.NoError(ct.Start())
	t.True(xerrors.Is(ct.Start(), util.DaemonAlreadyStartedError))

	time.Sleep(time.Millisecond * 100)
	t.True(atomic.LoadInt64(&ticked) < 4)
}

func (t *testCallbackTimer) TestStoppedByError() {
	var ticked int64
	ct, err := NewCallbackTimer(
		"good timer",
		func() (bool, error) {
			atomic.AddInt64(&ticked, 1)

			if atomic.LoadInt64(&ticked) < 2 {
				return true, nil
			}

			return true, xerrors.Errorf("idontknow")
		},
		time.Millisecond*10,
		nil,
	)
	t.NoError(err)

	t.NoError(ct.Start())
	t.True(xerrors.Is(ct.Start(), util.DaemonAlreadyStartedError))

	time.Sleep(time.Millisecond * 100)
	t.True(atomic.LoadInt64(&ticked) < 4)
}

func (t *testCallbackTimer) TestIntervalFunc() {
	var ticked int64
	ct, err := NewCallbackTimer(
		"good timer",
		func() (bool, error) {
			atomic.AddInt64(&ticked, 1)

			return true, nil
		},
		0,
		func() time.Duration {
			return time.Millisecond * 10
		},
	)
	t.NoError(err)

	t.NoError(ct.Start())
	t.True(xerrors.Is(ct.Start(), util.DaemonAlreadyStartedError))

	time.Sleep(time.Millisecond * 50)

	ct.Stop()

	t.True(atomic.LoadInt64(&ticked) > 3)
}

func (t *testCallbackTimer) TestIntervalFuncNarrowInterval() {
	var ticked int64
	ct, err := NewCallbackTimer(
		"good timer",
		func() (bool, error) {
			atomic.AddInt64(&ticked, 1)

			return true, nil
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
	t.True(xerrors.Is(ct.Start(), util.DaemonAlreadyStartedError))

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
			func() (bool, error) {
				atomic.AddInt64(ticked, 1)

				return true, nil
			},
			time.Millisecond*10,
			nil,
		)
		t.NoError(err)
		timers = append(timers, ct)
	}
	cts := NewCallbackTimerset(timers)
	t.NoError(cts.Start())

	<-time.After(time.Millisecond * 100)
	t.NoError(cts.Stop())

	for _, ticked := range tickeds {
		t.True(atomic.LoadInt64(ticked) > 3)
	}
}

func (t *testCallbackTimer) TestLongInterval() {
	ct, err := NewCallbackTimer(
		"long-interval timer",
		func() (bool, error) {
			return true, nil
		},
		time.Second*30,
		nil,
	)
	t.NoError(err)
	t.NoError(ct.Start())

	defer func() {
		t.NoError(ct.Stop())
	}()

	select {
	case <-time.After(time.Millisecond * 100):
		t.Error(xerrors.Errorf("stopping too long waited"))
	}
}

func TestCallbackTimer(t *testing.T) {
	suite.Run(t, new(testCallbackTimer))
}
