package mitum

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type emptyDaemon struct {
	*Daemon
}

func (ed *emptyDaemon) Start() error {
	if err := ed.Daemon.Start(); err != nil {
		return err
	}

	return nil
}

type testDaemon struct {
	suite.Suite
}

func (t *testDaemon) TestStart() {
	ed := &emptyDaemon{Daemon: NewDaemon(func(stopChan chan struct{}) error {
	end:
		for {
			select {
			case <-stopChan:
				break end
			default:
				time.Sleep(time.Millisecond * 100)
			}
		}

		return nil
	})}
	t.NoError(ed.Start())
	t.True(ed.IsStarted())

	// start again
	t.True(xerrors.Is(ed.Start(), DaemonAlreadyStartedError))

	defer func() {
		_ = ed.Stop()
	}()
}

func (t *testDaemon) TestStop() {
	ed := &emptyDaemon{Daemon: NewDaemon(func(stopChan chan struct{}) error {
	end:
		for {
			select {
			case <-stopChan:
				break end
			default:
				time.Sleep(time.Millisecond * 100)
			}
		}

		return nil
	})}
	t.NoError(ed.Start())
	t.True(ed.IsStarted())

	time.Sleep(time.Millisecond * 300)
	t.NoError(ed.Stop())
	t.True(ed.IsStopped())

	// stop again
	t.True(xerrors.Is(ed.Stop(), DaemonAlreadyStoppedError))
}

func (t *testDaemon) TestFunctionError() {
	ed := &emptyDaemon{Daemon: NewDaemon(func(stopChan chan struct{}) error {
		return xerrors.Errorf("find me :)")
	})}
	t.NoError(ed.Start())

	time.Sleep(time.Millisecond * 100)

	t.False(ed.IsStarted())
	t.True(xerrors.Is(ed.Stop(), DaemonAlreadyStoppedError))
}

func (t *testDaemon) TestStopByStopChan() {
	ed := &emptyDaemon{Daemon: NewDaemon(func(stopChan chan struct{}) error {
	end:
		for {
			select {
			case <-stopChan:
				break end
			}
		}

		return nil
	})}
	t.NoError(ed.Start())
	t.True(ed.IsStarted())

	time.Sleep(time.Millisecond * 50)

	ed.stopChan <- struct{}{}

	time.Sleep(time.Millisecond * 50)

	t.True(xerrors.Is(ed.Stop(), DaemonAlreadyStoppedError))
}

func (t *testDaemon) TestTimer() {
	var ticked int
	var wg sync.WaitGroup
	wg.Add(1)

	timer := &emptyDaemon{Daemon: NewDaemon(func(stopChan chan struct{}) error {
		ticker := time.NewTicker(time.Millisecond * 10)
		done := make(chan struct{})

		go func() {
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					ticked += 1
				}
			}
		}()

		<-stopChan
		done <- struct{}{}
		wg.Done()

		return nil
	})}
	t.NoError(timer.Start())
	t.True(timer.IsStarted())

	time.Sleep(time.Millisecond * 40)
	t.NoError(timer.Stop())

	wg.Wait()

	t.True(ticked > 2)
}

func (t *testDaemon) TestMultipleTimer() {
	n := 3

	var ticked int64
	var wgStopped, wgStarted sync.WaitGroup

	wgStopped.Add(n)
	wgStarted.Add(n)

	var timers []*emptyDaemon

	for i := 0; i < n; i++ {
		tr := &emptyDaemon{Daemon: NewDaemon(func(stopChan chan struct{}) error {
			ticker := time.NewTicker(time.Millisecond * 10)
			done := make(chan struct{})

			go func() {
				for {
					select {
					case <-done:
						return
					case <-ticker.C:
						atomic.AddInt64(&ticked, 1)
					}
				}
			}()

			<-stopChan
			done <- struct{}{}
			wgStopped.Done()

			return nil
		})}
		timers = append(timers, tr)

		go func() {
			t.NoError(tr.Start())
			t.True(tr.IsStarted())
			wgStarted.Done()
		}()
	}

	wgStarted.Wait()

	time.Sleep(time.Millisecond * 40)

	for _, tr := range timers {
		isTrue := t.True // NOTE testify suite occurs DATA RACE when called inside goroutine :(
		noError := t.NoError
		go func(ed *emptyDaemon) {
			noError(ed.Stop())
			isTrue(ed.IsStopped())
		}(tr)
	}

	wgStopped.Wait()

	t.True(atomic.LoadInt64(&ticked) > int64(n*2))
}

func TestDaemon(t *testing.T) {
	suite.Run(t, new(testDaemon))
}
