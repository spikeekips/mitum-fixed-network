package localtime

import (
	"testing"
	"time"

	"github.com/spikeekips/mitum/util"
	"github.com/stretchr/testify/suite"
)

type testTimers struct {
	suite.Suite
}

func (t *testTimers) timer(id TimerID) *ContextTimer {
	timer := NewContextTimer(
		id,
		time.Second*10,
		func(int) (bool, error) {
			return true, nil
		},
	)

	return timer
}

func (t *testTimers) TestStart() {
	ids := []TimerID{
		"showme",
	}

	timers := NewTimers(ids, false)
	t.NoError(timers.Start())
}

func (t *testTimers) TestAllowNew() {
	ids := []TimerID{
		"showme",
		"findme",
	}

	timers := NewTimers(ids, false)
	defer func() {
		_ = timers.Stop()
	}()

	id := TimerID("showme")
	t.NoError(timers.SetTimer(t.timer(id)))

	unknown := TimerID("unknown")
	t.Error(timers.SetTimer(t.timer(unknown)))
}

func (t *testTimers) TestStartTimer() {
	ids := []TimerID{
		"showme",
		"findme",
	}

	timers := NewTimers(ids, false)
	defer func() {
		_ = timers.Stop()
	}()

	for _, id := range ids {
		t.NoError(timers.SetTimer(t.timer(id)))
	}

	startID := TimerID("showme")
	stoppedID := TimerID("findme")

	t.NoError(timers.StartTimers([]TimerID{startID}, true))

	t.True(timers.timers[startID].IsStarted())
	t.Nil(timers.timers[stoppedID])
	t.True(timers.IsTimerStarted(startID))
}

func (t *testTimers) TestStartTimerStopOthers() {
	ids := []TimerID{
		"showme",
		"findme",
		"eatme",
	}

	timers := NewTimers(ids, false)
	defer func() {
		_ = timers.Stop()
	}()

	for _, id := range ids {
		t.NoError(timers.SetTimer(t.timer(id)))
	}

	// start all
	t.NoError(timers.StartTimers(ids, true))

	// start again only one
	startID := TimerID("showme")
	t.NoError(timers.StartTimers([]TimerID{startID}, true))
	t.True(timers.IsTimerStarted(startID))

	for _, id := range ids {
		if id == startID {
			continue
		}
		t.Nil(timers.timers[id])
		t.False(timers.IsTimerStarted(id))
	}
}

func (t *testTimers) TestStartTimerNotStop() {
	ids := []TimerID{
		"showme",
		"findme",
		"eatme",
	}

	timers := NewTimers(ids, false)
	defer func() {
		_ = timers.Stop()
	}()

	for _, id := range ids {
		t.NoError(timers.SetTimer(t.timer(id)))
	}

	// start all except startID
	t.NoError(timers.StartTimers(ids, true))

	startID := TimerID("showme")
	t.NoError(timers.StopTimers([]TimerID{startID}))
	t.Nil(timers.timers[startID])

	t.NoError(timers.SetTimer(t.timer(startID)))
	t.NoError(timers.StartTimers([]TimerID{startID}, false))

	for _, id := range ids {
		t.True(timers.timers[id].IsStarted())
	}
}

func (t *testTimers) TestStopTimer() {
	ids := []TimerID{
		"showme",
		"findme",
		"eatme",
	}

	timers := NewTimers(ids, false)
	defer func() {
		_ = timers.Stop()
	}()

	for _, id := range ids {
		t.NoError(timers.SetTimer(t.timer(id)))
	}

	// start all
	t.NoError(timers.StartTimers(ids, true))

	for _, id := range ids {
		t.True(timers.timers[id].IsStarted())
	}

	stopID := TimerID("eatme")
	t.NoError(timers.StopTimers([]TimerID{stopID}))
	t.Nil(timers.timers[stopID])

	for _, id := range ids {
		if id == stopID {
			continue
		}

		t.True(timers.timers[id].IsStarted())
	}

	st := timers.Started()
	t.Equal(2, len(st))

	started := make([]string, len(timers.Started()))
	for i := range st {
		started[i] = st[i].String()
	}

	t.True(util.InStringSlice("showme", started))
	t.True(util.InStringSlice("findme", started))
}

func (t *testTimers) TestStopTimersAll() {
	ids := []TimerID{
		"showme",
		"findme",
		"eatme",
	}

	timers := NewTimers(ids, false)
	defer func() {
		_ = timers.Stop()
	}()

	for _, id := range ids {
		t.NoError(timers.SetTimer(t.timer(id)))
	}

	// start all
	t.NoError(timers.StartTimers(ids, true))

	for _, id := range ids {
		t.True(timers.timers[id].IsStarted())
	}

	t.NoError(timers.StopTimersAll())

	for _, id := range ids {
		t.Nil(timers.timers[id])
	}
}

func (t *testTimers) TestStop() {
	ids := []TimerID{
		"showme",
		"findme",
		"eatme",
	}

	timers := NewTimers(ids, false)
	defer func() {
		_ = timers.Stop()
	}()

	for _, id := range ids {
		t.NoError(timers.SetTimer(t.timer(id)))
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
