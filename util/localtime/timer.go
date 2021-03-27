package localtime

import (
	"time"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/errors"
)

var (
	StopTimerError       = errors.NewError("stop timer")
	defaultTimerDuration = time.Hour * 24 * 360
)

type TimerID string

func (ti TimerID) String() string {
	return string(ti)
}

type Timer interface {
	util.Daemon
	IsStarted() bool
	ID() TimerID
	SetInterval(func(int) time.Duration) Timer
	Reset() error
}
