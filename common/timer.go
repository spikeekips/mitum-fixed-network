package common

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
)

const (
	TimerCallbackInvalidTimeoutErrorCode ErrorCode = iota + 1
	TimerCallbackChainAlreadyAddedErrorCode
	InvalidTimerCallbackForChainErrorCode
)

var (
	TimerCallbackInvalidTimeoutError Error = NewError(
		"timer", TimerCallbackInvalidTimeoutErrorCode, "invalid timeout value",
	)
	TimerCallbackChainAlreadyAddedError Error = NewError(
		"timer", TimerCallbackChainAlreadyAddedErrorCode, "callback timer already added",
	)
	InvalidTimerCallbackForChainError Error = NewError(
		"timer", InvalidTimerCallbackForChainErrorCode, "invalid callback timer for chain",
	)
)

type Timer interface {
	Daemon
	RunCount() uint
}

type TimerCallback func(Timer) error
type TimerCallbackIntervalFunc func(uint /* ran count */, time.Duration /* elapsed time */) time.Duration

type CallbackTimer struct {
	sync.RWMutex
	*Logger
	id           string
	name         string
	callbacks    []TimerCallback
	intervalFunc TimerCallbackIntervalFunc
	startedAt    Time
	runCount     uint
	limit        uint
	stopped      bool
	stopChan     chan struct{}
}

func NewCallbackTimer(name string, interval time.Duration, callbacks ...TimerCallback) *CallbackTimer {
	id := RandomUUID()
	ct := &CallbackTimer{
		Logger: NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.
				Str("module", name).
				Str("timer_id", id)
		}),
		id:        id,
		name:      name,
		callbacks: callbacks,
		intervalFunc: func(uint, time.Duration) time.Duration {
			return interval
		},
		stopChan: make(chan struct{}, 1),
	}

	return ct
}

func (ct *CallbackTimer) Name() string {
	return ct.name
}

func (ct *CallbackTimer) Start() error {
	ct.Lock()
	defer ct.Unlock()

	ct.startedAt = Now()
	ct.runCount = 0
	ct.stopped = false
	ct.stopChan = make(chan struct{}, 1)

	go ct.run()

	ct.Log().Debug().Msg("timer started")

	return nil
}

func (ct *CallbackTimer) Stop() error {
	if ct.IsStopped() {
		return nil
	}

	ct.Lock()
	defer ct.Unlock()

	ct.stopped = true
	ct.stopChan <- struct{}{}
	//close(ct.stopChan)
	ct.Log().Debug().Msg("timer stopped")

	return nil
}

func (ct *CallbackTimer) IsStopped() bool {
	ct.RLock()
	defer ct.RUnlock()

	return ct.stopped
}

func (ct *CallbackTimer) RunCount() uint {
	ct.RLock()
	defer ct.RUnlock()

	return ct.runCount
}

func (ct *CallbackTimer) Limit() uint {
	ct.RLock()
	defer ct.RUnlock()

	return ct.limit
}

func (ct *CallbackTimer) SetLimit(limit uint) *CallbackTimer {
	ct.Lock()
	defer ct.Unlock()

	ct.limit = ct.runCount + limit

	return ct
}

func (ct *CallbackTimer) incRunCount() {
	ct.Lock()
	defer ct.Unlock()

	ct.runCount++
}

func (ct *CallbackTimer) SetIntervalFunc(intervalFunc TimerCallbackIntervalFunc) *CallbackTimer {
	ct.Lock()
	defer ct.Unlock()

	ct.intervalFunc = intervalFunc

	return ct
}

func (ct *CallbackTimer) run() {
	if ct.IsStopped() {
		return
	}

	interval := ct.intervalFunc(ct.RunCount(), Now().Sub(ct.startedAt))
	if interval == 0 { // stop it
		return
	}

	select {
	case <-ct.stopChan:
		return
	case <-time.After(interval):
	}

	if ct.IsStopped() {
		return
	}

	if err := ct.runCallback(); err != nil {
		ct.Log().Error().Err(err).Msg("failed to run callback")
	}

	go ct.run()
}

func (ct *CallbackTimer) runCallback() error {
	limit := ct.Limit()
	runCount := ct.RunCount()

	if limit > 0 && runCount >= limit {
		ct.Log().Debug().
			Uint("count", runCount).
			Uint("limit", limit).
			Msg("reached to limit")
		return ct.Stop()
	}

	var wg sync.WaitGroup
	wg.Add(len(ct.callbacks))

	for i, callback := range ct.callbacks {
		go func(index int, cb TimerCallback) {
			defer wg.Done()

			if err := cb(ct); err != nil {
				ct.Log().Error().
					Err(err).
					Int("index", index).
					Msg("callback error")
			}
		}(i, callback)
	}

	wg.Wait()

	ct.incRunCount()

	return nil
}
