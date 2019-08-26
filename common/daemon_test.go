package common

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testReaderDaemon struct {
	suite.Suite
}

func (t *testReaderDaemon) TestNew() {
	count := 10
	var wg sync.WaitGroup
	wg.Add(count)

	callback := func(interface{}) error {
		defer wg.Done()

		return nil
	}

	d := NewReaderDaemon(true, 0, callback)
	d.SetLogger(zlog)

	err := d.Start()
	t.NoError(err)

	for i := 0; i < count; i++ {
		d.Write(1)
	}

	wg.Wait()

	err = d.Stop()
	t.NoError(err)

	after := time.After(time.Second * 2)
end:
	for {
		select {
		case <-after:
			t.NoError(xerrors.Errorf("not stopped"))
			break end
		default:
			if d.IsStopped() {
				break end
			}
		}
	}
}

func (t *testReaderDaemon) TestCount() {
	limit := 10
	var wg sync.WaitGroup
	wg.Add(limit)

	var sum uint64
	callback := func(v interface{}) error {
		defer wg.Done()

		atomic.AddUint64(&sum, uint64(v.(int)))

		return nil
	}

	d := NewReaderDaemon(true, 0, callback)
	d.SetLogger(zlog)

	err := d.Start()
	t.NoError(err)

	for i := 0; i < limit; i++ {
		d.Write(i)
	}

	wg.Wait()

	sumed := atomic.LoadUint64(&sum)
	t.Equal(45, int(sumed))

	err = d.Stop()
	t.NoError(err)
}

func (t *testReaderDaemon) TestAsynchronous() {
	limit := 10
	var wg sync.WaitGroup
	wg.Add(limit)

	var sum uint64
	callback := func(v interface{}) error {
		atomic.AddUint64(&sum, uint64(v.(int)))
		defer wg.Done()

		return nil
	}

	d := NewReaderDaemon(false, 0, callback)
	d.SetLogger(zlog)

	err := d.Start()
	t.NoError(err)

	for i := 0; i < limit; i++ {
		d.Write(i)
	}

	wg.Wait()

	sumed := atomic.LoadUint64(&sum)

	t.Equal(45, int(sumed))

	err = d.Stop()
	t.NoError(err)
}

func (t *testReaderDaemon) TestErrCallback() {
	limit := 10
	var wg sync.WaitGroup
	wg.Add(4)

	var sum uint64
	callback := func(v interface{}) error {
		if v.(int)%3 == 0 {
			return xerrors.Errorf("%d", v)
		}

		return nil
	}

	d := NewReaderDaemon(false, 0, callback)
	d.SetLogger(zlog)

	d.SetErrCallback(func(err error) {
		defer wg.Done()

		v, _ := strconv.ParseUint(err.Error(), 10, 64)
		atomic.AddUint64(&sum, uint64(v))
	})

	err := d.Start()
	t.NoError(err)

	for i := 0; i < limit; i++ {
		d.Write(i)
	}

	wg.Wait()

	sumed := atomic.LoadUint64(&sum)

	t.Equal(18, int(sumed))

	err = d.Stop()
	t.NoError(err)
}

func (t *testReaderDaemon) TestRestart() {
	limit := 10
	var wg sync.WaitGroup
	wg.Add(limit)

	var sum uint64
	callback := func(v interface{}) error {
		defer wg.Done()

		atomic.AddUint64(&sum, uint64(v.(int)))

		return nil
	}

	d := NewReaderDaemon(true, 0, callback)
	d.SetLogger(zlog)

	err := d.Start()
	t.NoError(err)

	for i := 0; i < limit; i++ {
		d.Write(i)
	}

	wg.Wait()

	{
		sumed := atomic.LoadUint64(&sum)
		t.Equal(45, int(sumed))

		err = d.Stop()
		t.NoError(err)
	}

	// restart
	<-time.After(time.Millisecond * 1)

	err = d.Start()
	t.NoError(err)

	wg.Add(limit)
	for i := 0; i < limit; i++ {
		d.Write(i)
	}

	wg.Wait()

	{
		sumed := atomic.LoadUint64(&sum)
		t.Equal(90, int(sumed))

		err = d.Stop()
		t.NoError(err)
	}
}

func TestReaderDaemon(t *testing.T) {
	suite.Run(t, new(testReaderDaemon))
}
