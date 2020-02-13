package localtime

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/util"
	"golang.org/x/xerrors"
)

type CallbackTimer struct {
	*logging.Logger
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
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.
				Str("module", "callback-timer").
				Str("name", name)
		}),
		name:         name,
		intervalFunc: intervalFunc,
	}
	ct.FunctionDaemon = util.NewFunctionDaemon(ct.callback(callback), true)

	return ct, nil
}

func (ct *CallbackTimer) SetLogger(l zerolog.Logger) *logging.Logger {
	_ = ct.Logger.SetLogger(l)
	_ = ct.FunctionDaemon.SetLogger(l)

	return ct.Logger
}

func (ct *CallbackTimer) Start() error {
	ct.Log().Debug().Msg("trying to start")
	defer ct.Log().Debug().Msg("timer started")

	return ct.FunctionDaemon.Start()
}

func (ct *CallbackTimer) Stop() error {
	ct.Log().Debug().Msg("trying to stop")
	defer ct.Log().Debug().Msg("timer stopped")

	return ct.FunctionDaemon.Stop()
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

type CallbackTimerset struct {
	timers []*CallbackTimer
}

func NewCallbackTimerset(timers []*CallbackTimer) *CallbackTimerset {
	return &CallbackTimerset{
		timers: timers,
	}
}

func (ct *CallbackTimerset) Start() error {
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

	return nil
}

func (ct *CallbackTimerset) Stop() error {
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

	close(errChan)

	wg.Wait()

	var err error
	for err = range errChan {
		if err != nil {
			break
		}
	}

	return err
}
