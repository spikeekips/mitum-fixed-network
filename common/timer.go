package common

import (
	"sync"
	"time"
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
}

type TimerCallback func(Timer) error
type TimerCallbackIntervalFunc func( /* ran count */ uint /* elapsed time */, time.Duration) time.Duration

type CallbackTimer struct {
	sync.RWMutex
	*Logger
	id           string
	daemon       *ReaderDaemon
	callbacks    []TimerCallback
	intervalFunc TimerCallbackIntervalFunc
	startedAt    Time
	runCount     uint
}

func NewCallbackTimer(name string, interval time.Duration, callbacks ...TimerCallback) *CallbackTimer {
	id := RandomUUID()
	ct := &CallbackTimer{
		id:        id,
		Logger:    NewLogger(Log(), "name", name, "timer_id", id),
		callbacks: callbacks,
		intervalFunc: func(uint, time.Duration) time.Duration {
			return interval
		},
	}
	ct.daemon = NewReaderDaemon(true, 0, ct.runCallback)
	_ = ct.daemon.SetLogContext(ct.LogContext())

	return ct
}

func (ct *CallbackTimer) Start() error {
	ct.Lock()
	defer ct.Unlock()

	if !ct.daemon.IsStopped() {
		return DaemonAleadyStartedError.Newf(
			"Timer is already running; daemon is still running; id=%q",
			ct.id,
		)
	}

	if err := ct.daemon.Start(); err != nil {
		return err
	}

	ct.startedAt = Now()
	ct.runCount = 0

	go ct.next()

	ct.Log().Debug("timer started")

	return nil
}

func (ct *CallbackTimer) Stop() error {
	ct.Lock()
	defer ct.Unlock()

	if err := ct.daemon.Stop(); err != nil {
		return err
	}
	ct.Log().Debug("timer stopped")

	return nil
}

func (ct *CallbackTimer) IsStopped() bool {
	return ct.daemon.IsStopped()
}

func (ct *CallbackTimer) SetIntervalFunc(intervalFunc TimerCallbackIntervalFunc) *CallbackTimer {
	ct.Lock()
	defer ct.Unlock()

	ct.intervalFunc = intervalFunc

	return ct
}

func (ct *CallbackTimer) RunCount() uint {
	ct.RLock()
	defer ct.RUnlock()

	return ct.runCount
}

func (ct *CallbackTimer) incRunCount() {
	ct.Lock()
	defer ct.Unlock()

	ct.runCount++
}

func (ct *CallbackTimer) runCallback(interface{}) error {
	var wg sync.WaitGroup
	wg.Add(len(ct.callbacks))

	for i, callback := range ct.callbacks {
		go func(index int, cb TimerCallback) {
			defer wg.Done()

			if err := cb(ct); err != nil {
				ct.Log().Error("callback error", "index", index, "error", err)
			}
		}(i, callback)
	}

	wg.Wait()

	ct.incRunCount()

	if ct.IsStopped() {
		return nil
	}

	go ct.next()

	return nil
}

func (ct *CallbackTimer) next() {
	<-time.After(ct.intervalFunc(ct.RunCount(), Now().Sub(ct.startedAt)))

	ct.daemon.Write(nil)
}
