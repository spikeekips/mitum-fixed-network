package localtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/util"
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

	t.NoError(timers.StartTimers([]string{startID}, true))

	t.True(timers.timers[startID].IsStarted())
	t.Nil(timers.timers[stoppedID])
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
	t.NoError(timers.StartTimers(ids, true))

	// start again only one
	startID := "showme"
	t.NoError(timers.StartTimers([]string{startID}, true))

	for _, id := range ids {
		if id == startID {
			continue
		}
		t.Nil(timers.timers[id])
	}
}

func (t *testTimers) TestStartTimerNotStop() {
	ids := []string{
		"showme",
		"findme",
		"eatme",
	}

	timers := NewTimers(ids, false)

	for _, id := range ids {
		t.NoError(timers.SetTimer(id, t.timer(id)))
	}

	// start all except startID
	t.NoError(timers.StartTimers(ids, true))

	startID := "showme"
	t.NoError(timers.StopTimers([]string{startID}))
	t.Nil(timers.timers[startID])

	t.NoError(timers.SetTimer(startID, t.timer(startID)))
	t.NoError(timers.StartTimers([]string{startID}, false))

	for _, id := range ids {
		t.True(timers.timers[id].IsStarted())
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
	t.NoError(timers.StartTimers(ids, true))

	for _, id := range ids {
		t.True(timers.timers[id].IsStarted())
	}

	stopID := "eatme"
	t.NoError(timers.StopTimers([]string{stopID}))
	t.Nil(timers.timers[stopID])

	for _, id := range ids {
		if id == stopID {
			continue
		}

		t.True(timers.timers[id].IsStarted())
	}

	t.Equal(2, len(timers.Started()))
	t.True(util.InStringSlice("showme", timers.Started()))
	t.True(util.InStringSlice("findme", timers.Started()))
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
	t.NoError(timers.StartTimers(ids, true))

	for _, id := range ids {
		t.True(timers.timers[id].IsStarted())
	}

	t.NoError(timers.Stop())

	for _, id := range ids {
		t.Nil(timers.timers[id])
	}
}

func TestTimers(t *testing.T) {
	suite.Run(t, new(testTimers))
}
