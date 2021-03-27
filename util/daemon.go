package util

import (
	"github.com/spikeekips/mitum/util/errors"
)

var (
	DaemonAlreadyStartedError = errors.NewError("daemon already started")
	DaemonAlreadyStoppedError = errors.NewError("daemon already stopped")
)

type Daemon interface {
	Start() error
	Stop() error
}
