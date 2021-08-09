package util

var (
	DaemonAlreadyStartedError = NewError("daemon already started")
	DaemonAlreadyStoppedError = NewError("daemon already stopped")
)

type Daemon interface {
	Start() error
	Stop() error
}
