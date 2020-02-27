package localtime

import (
	"sync"

	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/util"
	"golang.org/x/xerrors"
)

// Timers handles the multiple timers and controls them selectively.
// TODO Timers will replace CallbackTimerset
type Timers struct {
	*logging.Logger
	sync.RWMutex
	timers   map[ /* timer id */ string]*CallbackTimer
	allowNew bool // if allowNew is true, new timer can be added.
}

func NewTimers(ids []string, allowNew bool) *Timers {
	timers := map[string]*CallbackTimer{}
	for _, id := range ids {
		timers[id] = nil
	}

	return &Timers{timers: timers, allowNew: allowNew}
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
		t := ts.timers[id]
		go func(t *CallbackTimer) {
			defer wg.Done()

			if err := t.Stop(); err != nil {
				ts.Log().Error().Err(err).Str("timer", t.Name()).Msg("failed to stop timer")
			}
		}(t)
	}

	wg.Wait()

	return nil
}

// SetTimer sets the timer with id
func (ts *Timers) SetTimer(id string, timer *CallbackTimer) error {
	ts.Lock()
	defer ts.Unlock()

	if _, found := ts.timers[id]; !found {
		if !ts.allowNew {
			return xerrors.Errorf("not allowed to add new timer: %s", id)
		}
	}

	existing := ts.timers[id]
	if existing != nil && existing.IsStarted() {
		if err := existing.Stop(); err != nil {
			return err
		}
	}

	ts.timers[id] = timer

	return nil
}

// StartTimers starts timers with the given ids, before starting timers, stops
// the other timers if stopOthers is true.
func (ts *Timers) StartTimers(ids []string, stopOthers bool) error {
	ts.Lock()
	defer ts.Unlock()

	if stopOthers {
		var stopIDs []string
		for id := range ts.timers {
			if util.InStringSlice(id, ids) {
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
			ts.Log().Error().Err(err).Str("timer", t.Name()).Msg("failed to start timer")
		}
	}

	return ts.traverse(callback, ids)
}

func (ts *Timers) StopTimers(ids []string) error {
	ts.Lock()
	defer ts.Unlock()

	return ts.stopTimers(ids)
}

func (ts *Timers) stopTimers(ids []string) error {
	callback := func(t *CallbackTimer) {
		if t.IsStopped() {
			return
		}

		if err := t.Stop(); err != nil {
			ts.Log().Error().Err(err).Str("timer", t.Name()).Msg("failed to start timer")
		}
	}

	return ts.traverse(callback, ids)
}

func (ts *Timers) Started() []string {
	ts.RLock()
	defer ts.RUnlock()

	var started []string
	for id := range ts.timers {
		timer := ts.timers[id]
		if timer != nil && ts.timers[id].IsStarted() {
			started = append(started, id)
		}
	}

	return started
}

func (ts *Timers) checkExists(ids []string) error {
	for _, id := range ids {
		if _, found := ts.timers[id]; !found {
			return xerrors.Errorf("timer not found: %s", id)
		}
	}

	return nil
}

func (ts *Timers) traverse(callback func(*CallbackTimer), ids []string) error {
	if err := ts.checkExists(ids); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(len(ids))

	for _, id := range ids {
		go func(id string) {
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
