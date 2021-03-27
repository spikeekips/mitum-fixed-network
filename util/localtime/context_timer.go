package localtime

import (
	"context"
	"sync"
	"time"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

var contextTimerPool = sync.Pool{
	New: func() interface{} {
		return new(ContextTimer)
	},
}

var (
	ContextTimerPoolGet = func() *ContextTimer {
		return contextTimerPool.Get().(*ContextTimer)
	}
	ContextTimerPoolPut = func(ct *ContextTimer) {
		ct.Lock()
		ct.Logging = nil
		ct.ContextDaemon = nil
		ct.id = TimerID("")
		ct.interval = nil
		ct.callback = nil
		ct.runchan = nil
		ct.c = 0
		ct.Unlock()

		contextTimerPool.Put(ct)
	}
)

type ContextTimer struct {
	sync.RWMutex
	*logging.Logging
	*util.ContextDaemon
	id       TimerID
	interval func(int) time.Duration
	callback func(int) (bool, error)
	runchan  chan error
	c        int
}

func NewContextTimer(id TimerID, interval time.Duration, callback func(int) (bool, error)) *ContextTimer {
	ct := ContextTimerPoolGet()
	ct.Logging = logging.NewLogging(func(c logging.Context) logging.Emitter {
		return c.Str("module", "context-timer").
			Str("id", id.String())
	})
	ct.id = id
	ct.interval = func(int) time.Duration {
		return interval
	}
	ct.callback = callback
	ct.runchan = make(chan error, 1)
	ct.ContextDaemon = util.NewContextDaemon(string(id), ct.start)

	return ct
}

func (ct *ContextTimer) ID() TimerID {
	return ct.id
}

func (ct *ContextTimer) SetInterval(f func(int) time.Duration) Timer {
	ct.Lock()
	defer ct.Unlock()

	ct.interval = f

	return ct
}

func (ct *ContextTimer) Reset() error {
	ct.Lock()
	defer ct.Unlock()

	ct.c = 0

	return nil
}

func (ct *ContextTimer) Stop() error {
	if err := ct.ContextDaemon.Stop(); err != nil {
		return err
	}

	close(ct.runchan)

	return nil
}

func (ct *ContextTimer) count() int {
	ct.RLock()
	defer ct.RUnlock()

	return ct.c
}

func (ct *ContextTimer) start(ctx context.Context) error {
	if err := ct.Reset(); err != nil {
		return err
	}

	errchan := make(chan error)

	ct.runchan <- nil

end:
	for {
		select {
		case <-ctx.Done():
			break end
		case err := <-errchan:
			if !xerrors.Is(err, StopTimerError) {
				ct.Log().Debug().Err(err).Msg("timer got error; timer will be stopped")
			}

			break end
		case <-ct.runchan:
			go func(count int) {
				if err := ct.runCallback(ctx, count); err != nil {
					errchan <- err
				}
			}(ct.count())
		}
	}

	return nil
}

func (ct *ContextTimer) runCallback(ctx context.Context, count int) error {
	ct.Lock()
	defer ct.Unlock()

	var interval time.Duration
	if i := ct.interval(count); i < time.Nanosecond {
		return xerrors.Errorf("invalid interval; too narrow, %v", i)
	} else {
		interval = i
	}

	select {
	case <-ctx.Done():
		return nil
	case <-time.After(interval):
	}

	if keep, err := ct.callback(count); err != nil {
		return err
	} else if !keep {
		return StopTimerError
	}

	if ct.c == count {
		ct.c++
	}
	ct.runchan <- nil

	return nil
}
