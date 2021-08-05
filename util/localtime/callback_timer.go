package localtime

import (
	"sync"
	"time"

	"golang.org/x/xerrors"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

type CallbackTimer struct {
	sync.RWMutex
	*logging.Logging
	id           TimerID
	callback     func(int) (bool, error)
	intervalFunc func(int) time.Duration
	errchan      chan error
	ticker       *time.Ticker
	stopped      bool
	stopChan     chan struct{}
	resetChan    chan struct{}
}

func NewCallbackTimer(
	id TimerID,
	callback func(int) (bool, error),
	interval time.Duration,
) (*CallbackTimer, error) {
	return &CallbackTimer{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "next-callback-timer").Stringer("id", id)
		}),
		id: id,
		intervalFunc: func(int) time.Duration {
			return interval
		},
		callback:  callback,
		errchan:   make(chan error, 100),
		stopped:   true,
		stopChan:  make(chan struct{}, 1),
		ticker:    time.NewTicker(defaultTimerDuration),
		resetChan: make(chan struct{}),
	}, nil
}

func (ct *CallbackTimer) ID() TimerID {
	return ct.id
}

// SetInterval sets the interval function. If the returned duration is 0, the
// timer will be stopped.
func (ct *CallbackTimer) SetInterval(f func(int) time.Duration) Timer {
	ct.Lock()
	defer ct.Unlock()

	ct.intervalFunc = f

	return ct
}

func (ct *CallbackTimer) Start() error {
	ct.Lock()
	defer ct.Unlock()

	if i := ct.intervalFunc(0); i < time.Nanosecond {
		return xerrors.Errorf("too narrow interval: %v", i)
	}

	return ct.start()
}

func (ct *CallbackTimer) start() error {
	if !ct.stopped {
		return util.DaemonAlreadyStartedError
	}

	ct.stopped = false

	ct.ticker.Reset(defaultTimerDuration)
	ct.stopChan = make(chan struct{}, 1)

	go ct.clock()

	ct.Log().Debug().Msg("timer started")

	return nil
}

func (ct *CallbackTimer) Stop() error {
	ct.Lock()
	defer ct.Unlock()

	return ct.stop()
}

func (ct *CallbackTimer) stop() error {
	if ct.stopped {
		return nil
	}

	ct.stopped = true

	ct.stopChan <- struct{}{}

	ct.Log().Debug().Msg("timer stopped")

	return nil
}

func (ct *CallbackTimer) Restart() error {
	ct.Lock()
	defer ct.Unlock()

	if !ct.stopped {
		if err := ct.stop(); err != nil {
			return err
		}
	}

	return ct.start()
}

func (ct *CallbackTimer) Reset() error {
	ct.Lock()
	defer ct.Unlock()

	if ct.stopped {
		return nil
	}

	ct.resetChan <- struct{}{}

	return nil
}

func (ct *CallbackTimer) IsStarted() bool {
	ct.RLock()
	defer ct.RUnlock()

	return !ct.stopped
}

func (ct *CallbackTimer) clock() {
	lastInterval, err := ct.resetTicker(0, 0)
	if err != nil {
		_ = ct.Stop()

		return
	}

	defer ct.ticker.Stop()

	var i int

end:
	for {
		select {
		case <-ct.stopChan:
			return
		case <-ct.resetChan:
			i = 0
			j, err := ct.resetTicker(0, lastInterval)
			if err != nil {
				break end
			}
			lastInterval = j
		case err := <-ct.errchan:
			if err == nil {
				continue
			}

			if xerrors.Is(err, StopTimerError) {
				ct.Log().Debug().Msg("timer will be stopped by callback")
			} else {
				ct.Log().Error().Err(err).Msg("timer got error; timer will be stopped")
			}

			break end
		case <-ct.ticker.C:
			go func(i int) {
				if keep, err := ct.callback(i); err != nil {
					ct.errchan <- err
				} else if !keep {
					ct.errchan <- StopTimerError
				}
			}(i)

			i++

			d, err := ct.resetTicker(i, lastInterval)
			if err != nil {
				break end
			}
			lastInterval = d
		}
	}

	_ = ct.Stop()
}

func (ct *CallbackTimer) resetTicker(i int, last time.Duration) (time.Duration, error) {
	in := ct.intervalFunc(i)
	if in < time.Nanosecond {
		return 0, xerrors.Errorf("too narrow interval: %v", in)
	}
	if in != last {
		ct.ticker.Reset(in)
	}

	return in, nil
}
