package localtime

import (
	"sync"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/util"
)

type CallbackTimer struct {
	*logging.Logging
	*util.FunctionDaemon
	name         string
	intervalFunc func() time.Duration
}

func NewCallbackTimer(
	name string,
	callback func() (bool, error),
	defaultInterval time.Duration,
	intervalFunc func() time.Duration,
) (*CallbackTimer, error) {
	if defaultInterval < 1 && intervalFunc == nil {
		return nil, xerrors.Errorf("interval is missing")
	}

	if intervalFunc == nil {
		intervalFunc = func() time.Duration {
			return defaultInterval
		}
	}

	ct := &CallbackTimer{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "callback-timer").
				Str("name", name)
		}),
		name:         name,
		intervalFunc: intervalFunc,
	}
	ct.FunctionDaemon = util.NewFunctionDaemon(ct.callback(callback), false)

	return ct, nil
}

func (ct *CallbackTimer) SetLogger(l logging.Logger) logging.Logger {
	_ = ct.Logging.SetLogger(l)
	_ = ct.FunctionDaemon.SetLogger(l)

	return ct.Log()
}

func (ct *CallbackTimer) Start() error {
	ct.Log().Debug().Msg("trying to start")
	defer ct.Log().Debug().Msg("timer started")

	return ct.FunctionDaemon.Start()
}

func (ct *CallbackTimer) Stop() error {
	ct.Log().Debug().Msg("trying to stop")
	defer ct.Log().Debug().Msg("timer stopped")

	err := ct.FunctionDaemon.Stop()
	if xerrors.Is(err, util.DaemonAlreadyStoppedError) {
		return nil
	}

	return err
}

func (ct *CallbackTimer) callback(cb func() (bool, error)) func(chan struct{}) error {
	return func(stopChan chan struct{}) error {
		returnChan := make(chan error)

		i := ct.intervalFunc()
		if i < time.Nanosecond {
			return xerrors.Errorf("too narrow interval: %v", i)
		}
		ticker := time.NewTicker(i)
		defer ticker.Stop()

		go func() {
			errChan := make(chan error)
			for {
				select {
				case err := <-errChan:
					returnChan <- err
					return
				case <-stopChan:
					returnChan <- nil
					return
				case <-ticker.C:
					go func() {
						if keep, err := cb(); err != nil {
							errChan <- err
						} else if !keep {
							errChan <- xerrors.Errorf("don't go")
						}
					}()

					i := ct.intervalFunc()
					if i < time.Nanosecond {
						returnChan <- xerrors.Errorf("too narrow interval: %v", i)
						return
					}

					ticker = time.NewTicker(i)
				}
			}
		}()

		return <-returnChan
	}
}

func (ct *CallbackTimer) Name() string {
	return ct.name
}

type CallbackTimerset struct {
	sync.RWMutex
	timers    []*CallbackTimer
	isStarted bool
}

func NewCallbackTimerset(timers []*CallbackTimer) *CallbackTimerset {
	return &CallbackTimerset{
		timers: timers,
	}
}

func (ct *CallbackTimerset) SetLogger(l logging.Logger) logging.Logger {
	for _, t := range ct.timers {
		_ = t.SetLogger(l)
	}

	return logging.Logger{}
}

func (ct *CallbackTimerset) Start() error {
	ct.Lock()
	defer ct.Unlock()

	var wg sync.WaitGroup
	wg.Add(len(ct.timers))

	errChan := make(chan error, len(ct.timers))
	for _, tr := range ct.timers {
		if !tr.IsStopped() {
			wg.Done()
			continue
		}

		go func(t *CallbackTimer) {
			if err := t.Start(); err != nil {
				errChan <- err
			}
			wg.Done()
		}(tr)
	}

	close(errChan)

	wg.Wait()

	var err error
	for err = range errChan {
		if err != nil {
			break
		}
	}

	if err != nil {
		wg.Add(len(ct.timers))

		// stop started timer
		for _, tr := range ct.timers {
			if !tr.IsStarted() {
				wg.Done()
				continue
			}

			go func(t *CallbackTimer) {
				_ = t.Stop()
				wg.Done()
			}(tr)
		}
		wg.Wait()

		return err
	}

	ct.isStarted = true

	return nil
}

func (ct *CallbackTimerset) Stop() error {
	if !ct.IsStarted() {
		return nil
	}

	ct.Lock()
	defer ct.Unlock()

	var wg sync.WaitGroup
	wg.Add(len(ct.timers))

	errChan := make(chan error, len(ct.timers))
	for _, tr := range ct.timers {
		if !tr.IsStarted() {
			wg.Done()
			continue
		}

		go func(t *CallbackTimer) {
			if err := t.Stop(); err != nil {
				errChan <- err
			}
			wg.Done()
		}(tr)
	}

	wg.Wait()
	close(errChan)

	var err error
	for err = range errChan {
		if err != nil {
			break
		}
	}

	ct.isStarted = false

	return err
}

func (ct *CallbackTimerset) IsStarted() bool {
	ct.RLock()
	defer ct.RUnlock()

	return ct.isStarted
}
