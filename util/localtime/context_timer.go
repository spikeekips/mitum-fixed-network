package localtime

import (
	"context"
	"sync"
	"time"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
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
	ct := &ContextTimer{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "context-timer").
				Str("id", id.String())
		}),
		id: id,
		interval: func(int) time.Duration {
			return interval
		},
		callback: callback,
		runchan:  make(chan error, 1),
	}

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

func (ct *ContextTimer) count() int {
	ct.RLock()
	defer ct.RUnlock()

	return ct.c
}

func (ct *ContextTimer) incCount(count int) {
	ct.Lock()
	defer ct.Unlock()

	if count != ct.c {
		return
	}

	ct.c++
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

	ct.incCount(count)
	ct.runchan <- nil

	return nil
}
