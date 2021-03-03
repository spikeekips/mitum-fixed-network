package localtime

import (
	"sync"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

// Timers handles the multiple timers and controls them selectively.
type Timers struct {
	*logging.Logging
	sync.RWMutex
	timers   map[ /* timer id */ TimerID]*CallbackTimer
	allowNew bool // if allowNew is true, new timer can be added.
}

func NewTimers(ids []TimerID, allowNew bool) *Timers {
	timers := map[TimerID]*CallbackTimer{}
	for _, id := range ids {
		timers[id] = nil
	}

	return &Timers{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "timers")
		}),
		timers:   timers,
		allowNew: allowNew,
	}
}

func (ts *Timers) SetLogger(l logging.Logger) logging.Logger {
	ts.Lock()
	defer ts.Unlock()

	_ = ts.Logging.SetLogger(l)

	for id := range ts.timers {
		timer := ts.timers[id]
		if timer == nil {
			continue
		}

		_ = timer.SetLogger(l)
	}

	return ts.Log()
}

// Start of Timers does nothing
func (ts *Timers) Start() error {
	return nil
}

// Stop of Timers will stop all the timers
func (ts *Timers) Stop() error {
	ts.Lock()
	defer ts.Unlock()

	var wg sync.WaitGroup
	wg.Add(len(ts.timers))

	for id := range ts.timers {
		timer := ts.timers[id]
		if timer == nil {
			wg.Done()
			continue
		}

		go func(t *CallbackTimer) {
			defer wg.Done()

			if err := t.Stop(); err != nil {
				ts.Log().Error().Err(err).Str("timer", t.ID().String()).Msg("failed to stop timer")
			}
		}(timer)
	}

	wg.Wait()

	for id := range ts.timers {
		ts.timers[id] = nil
	}

	return nil
}

func (ts *Timers) ResetTimer(id TimerID) error {
	ts.RLock()
	defer ts.RUnlock()

	switch t, found := ts.timers[id]; {
	case !found:
		return xerrors.Errorf("timer, %q not found", id)
	case t == nil:
		return xerrors.Errorf("timer, %q not running", id)
	default:
		return t.Reset()
	}
}

// SetTimer sets the timer with id
func (ts *Timers) SetTimer(timer *CallbackTimer) error {
	ts.Lock()
	defer ts.Unlock()

	if _, found := ts.timers[timer.ID()]; !found {
		if !ts.allowNew {
			return xerrors.Errorf("not allowed to add new timer: %s", timer.ID())
		}
	}

	existing := ts.timers[timer.ID()]
	if existing != nil && existing.IsStarted() {
		if err := existing.Stop(); err != nil {
			return err
		}
	}

	ts.timers[timer.ID()] = timer

	if timer != nil {
		_ = ts.timers[timer.ID()].SetLogger(ts.Log())
	}

	return nil
}

// StartTimers starts timers with the given ids, before starting timers, stops
// the other timers if stopOthers is true.
func (ts *Timers) StartTimers(ids []TimerID, stopOthers bool) error {
	ts.Lock()
	defer ts.Unlock()

	sids := make([]string, len(ids))
	for i := range ids {
		sids[i] = ids[i].String()
	}

	if stopOthers {
		var stopIDs []TimerID
		for id := range ts.timers {
			if util.InStringSlice(id.String(), sids) {
				continue
			}
			stopIDs = append(stopIDs, id)
		}

		if len(stopIDs) > 0 {
			if err := ts.stopTimers(stopIDs); err != nil {
				return err
			}
		}
	}

	callback := func(t *CallbackTimer) {
		if t.IsStarted() {
			return
		}

		if err := t.Start(); err != nil {
			ts.Log().Error().Err(err).Str("timer", t.ID().String()).Msg("failed to start timer")
		}
	}

	return ts.traverse(callback, ids)
}

func (ts *Timers) StopTimers(ids []TimerID) error {
	ts.Lock()
	defer ts.Unlock()

	return ts.stopTimers(ids)
}

func (ts *Timers) stopTimers(ids []TimerID) error {
	callback := func(t *CallbackTimer) {
		if !t.IsStarted() {
			return
		}

		if err := t.Stop(); err != nil {
			ts.Log().Error().Err(err).Str("timer", t.ID().String()).Msg("failed to start timer")
		}
	}

	if err := ts.traverse(callback, ids); err != nil {
		return err
	}

	for _, id := range ids {
		ts.timers[id] = nil
	}

	return nil
}

func (ts *Timers) Started() []TimerID {
	ts.RLock()
	defer ts.RUnlock()

	var started []TimerID
	for id := range ts.timers {
		timer := ts.timers[id]
		if timer != nil && ts.timers[id].IsStarted() {
			started = append(started, id)
		}
	}

	return started
}

func (ts *Timers) checkExists(ids []TimerID) error {
	for _, id := range ids {
		if _, found := ts.timers[id]; !found {
			return xerrors.Errorf("timer not found: %s", id)
		}
	}

	return nil
}

func (ts *Timers) traverse(callback func(*CallbackTimer), ids []TimerID) error {
	if err := ts.checkExists(ids); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(len(ids))

	for _, id := range ids {
		go func(id TimerID) {
			defer wg.Done()

			timer := ts.timers[id]
			if timer == nil {
				return
			}

			callback(timer)
		}(id)
	}

	wg.Wait()

	return nil
}
