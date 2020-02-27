package localtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type testTimers struct {
	suite.Suite
}

func (t *testTimers) timer(id string) *CallbackTimer {
	timer, err := NewCallbackTimer(
		id,
		func() (bool, error) {
			return true, nil
		},
		time.Second*10,
		nil,
	)
	t.NoError(err)

	return timer
}

func (t *testTimers) TestStart() {
	ids := []string{
		"showme",
	}

	timers := NewTimers(ids, false)
	t.NoError(timers.Start())
}

func (t *testTimers) TestAllowNew() {
	ids := []string{
		"showme",
		"findme",
	}

	timers := NewTimers(ids, false)

	id := "showme"
	t.NoError(timers.SetTimer(id, t.timer(id)))

	unknown := "unknown"
	t.Error(timers.SetTimer(unknown, t.timer(unknown)))
}

func (t *testTimers) TestStartTimer() {
	ids := []string{
		"showme",
		"findme",
	}

	timers := NewTimers(ids, false)

	for _, id := range ids {
		t.NoError(timers.SetTimer(id, t.timer(id)))
	}

	startID := "showme"
	stoppedID := "findme"

	t.NoError(timers.StartTimers([]string{startID}))

	t.True(timers.timers[startID].IsStarted())
	t.False(timers.timers[stoppedID].IsStarted())
}

func (t *testTimers) TestStartTimerStopOthers() {
	ids := []string{
		"showme",
		"findme",
		"eatme",
	}

	timers := NewTimers(ids, false)

	for _, id := range ids {
		t.NoError(timers.SetTimer(id, t.timer(id)))
	}

	// start all
	t.NoError(timers.StartTimers(ids))

	// start again only one
	startID := "showme"
	t.NoError(timers.StartTimers([]string{startID}))

	for _, id := range ids {
		if id == startID {
			continue
		}
		t.False(timers.timers[id].IsStarted())
	}
}

func (t *testTimers) TestStopTimer() {
	ids := []string{
		"showme",
		"findme",
		"eatme",
	}

	timers := NewTimers(ids, false)

	for _, id := range ids {
		t.NoError(timers.SetTimer(id, t.timer(id)))
	}

	// start all
	t.NoError(timers.StartTimers(ids))

	for _, id := range ids {
		t.True(timers.timers[id].IsStarted())
	}

	stopID := "eatme"
	t.NoError(timers.StopTimers([]string{stopID}))
	t.False(timers.timers[stopID].IsStarted())

	for _, id := range ids {
		if id == stopID {
			continue
		}

		t.True(timers.timers[id].IsStarted())
	}
}

func (t *testTimers) TestStopAll() {
	ids := []string{
		"showme",
		"findme",
		"eatme",
	}

	timers := NewTimers(ids, false)

	for _, id := range ids {
		t.NoError(timers.SetTimer(id, t.timer(id)))
	}

	// start all
	t.NoError(timers.StartTimers(ids))

	for _, id := range ids {
		t.True(timers.timers[id].IsStarted())
	}

	t.NoError(timers.Stop())

	for _, id := range ids {
		t.False(timers.timers[id].IsStarted())
	}
}

func TestTimers(t *testing.T) {
	suite.Run(t, new(testTimers))
}
