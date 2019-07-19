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
	callback := func(ti Timer) error {
		atomic.AddUint64(&runCount, 1)
		return nil
	}

	ct := NewCallbackTimer("test", time.Millisecond*2, callback)

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
	callback := func(ti Timer) error {
		atomic.AddUint64(&runCount, 1)
		return nil
	}

	timeout := time.Millisecond * 50
	defaultInterval := time.Millisecond * 1

	intervalFunc := func(index uint, elapsed time.Duration) time.Duration {
		if index > 2 {
			return timeout + time.Second*50
		}

		return defaultInterval
	}

	ct := NewCallbackTimer("test", defaultInterval, callback)
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
	callback := func(ti Timer) error {
		atomic.AddUint64(&runCount, 1)
		return nil
	}

	ct := NewCallbackTimer("test", time.Millisecond*2, callback, callback, callback)

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

func TestCallbackTimer(t *testing.T) {
	suite.Run(t, new(testCallbackTimer))
}
