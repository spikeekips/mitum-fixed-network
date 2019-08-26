package common

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type testCallbackTimer struct {
	suite.Suite
}

func (t *testCallbackTimer) TestNew() {
	var runCount uint64
	callback := func(Timer) error {
		atomic.AddUint64(&runCount, 1)
		return nil
	}

	ct := NewCallbackTimer("test", time.Millisecond*2, callback)
	ct.SetLogger(zlog)

	err := ct.Start()
	t.NoError(err)

	<-time.After(time.Millisecond * 50)
	err = ct.Stop()
	t.NoError(err)

	<-time.After(time.Millisecond * 50)

	runCounted := atomic.LoadUint64(&runCount)
	countedIntimer := uint64(ct.RunCount())
	t.True(runCounted > countedIntimer-1 && runCounted < countedIntimer+1)
	//t.Equal(runCounted, uint64(ct.RunCount()))
}

func (t *testCallbackTimer) TestIntervalFunc() {
	var runCount uint64
	callback := func(Timer) error {
		atomic.AddUint64(&runCount, 1)
		return nil
	}

	timeout := time.Millisecond * 50
	defaultInterval := time.Millisecond * 1

	intervalFunc := func(index uint, _ time.Duration) time.Duration {
		if index > 2 {
			return timeout + time.Second*50
		}

		return defaultInterval
	}

	ct := NewCallbackTimer("test", defaultInterval, callback)
	ct.SetLogger(zlog)
	ct.SetIntervalFunc(intervalFunc)

	err := ct.Start()
	t.NoError(err)

	<-time.After(timeout)
	err = ct.Stop()
	t.NoError(err)

	<-time.After(time.Millisecond * 50)

	runCounted := atomic.LoadUint64(&runCount)
	t.Equal(uint64(3), runCounted)

	countedIntimer := uint64(ct.RunCount())
	t.True(runCounted > countedIntimer-1 && runCounted < countedIntimer+1)
	//t.Equal(runCounted, uint64(ct.RunCount()))
}

func (t *testCallbackTimer) TestMultipleCallbacks() {
	var runCount uint64
	callback := func(Timer) error {
		atomic.AddUint64(&runCount, 1)
		return nil
	}

	ct := NewCallbackTimer("test", time.Millisecond*2, callback, callback, callback)
	ct.SetLogger(zlog)

	err := ct.Start()
	t.NoError(err)

	<-time.After(time.Millisecond * 50)
	err = ct.Stop()
	t.NoError(err)

	<-time.After(time.Millisecond * 50)

	runCounted := atomic.LoadUint64(&runCount)
	countedIntimer := uint64(ct.RunCount()) * 3
	t.True(runCounted > countedIntimer-3 && runCounted < countedIntimer+3)
	//t.Equal(runCounted, uint64(ct.RunCount())*3)
}

func (t *testCallbackTimer) TestLimit() {
	var runCount uint64
	callback := func(Timer) error {
		atomic.AddUint64(&runCount, 1)
		return nil
	}

	ct := NewCallbackTimer("test", time.Millisecond*1, callback)
	ct.SetLogger(zlog)
	defer ct.Stop()

	var limit uint = 3
	ct.SetLimit(limit)

	err := ct.Start()
	t.NoError(err)

	<-time.After(time.Millisecond * 10)
	t.NoError(err)

	runCounted := atomic.LoadUint64(&runCount)
	countedIntimer := uint64(ct.RunCount())

	t.Equal(uint64(limit), runCounted)
	t.Equal(uint64(limit), countedIntimer)
}

func (t *testCallbackTimer) TestZeroInterval() {
	var runCount uint64
	callback := func(Timer) error {
		atomic.AddUint64(&runCount, 1)
		return nil
	}

	timeout := time.Millisecond * 10
	interval := time.Millisecond * 1

	intervalFunc := func(_ uint, elapsed time.Duration) time.Duration {
		if elapsed >= timeout {
			return 0
		}

		return interval
	}

	ct := NewCallbackTimer("test", interval, callback)
	ct.SetLogger(zlog)
	ct.SetIntervalFunc(intervalFunc)

	err := ct.Start()
	t.NoError(err)

	<-time.After(timeout * 2)
	err = ct.Stop()
	t.NoError(err)

	<-time.After(time.Millisecond * 10)

	runCounted := atomic.LoadUint64(&runCount)
	t.True(uint64(timeout) > runCounted)
}

func TestCallbackTimer(t *testing.T) {
	suite.Run(t, new(testCallbackTimer))
}
