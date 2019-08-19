package common

import (
	"sync"
)

const (
	DaemonAleadyStartedErrorCode ErrorCode = iota + 1
	DaemonAleadyStoppedErrorCode
)

var (
	DaemonAleadyStartedError = NewError("daemon", DaemonAleadyStartedErrorCode, "daemon already started")
	DaemonAleadyStoppedError = NewError("daemon", DaemonAleadyStoppedErrorCode, "daemon already stopped")
)

type Daemon interface {
	Start() error
	Stop() error
	IsStopped() bool
}

type ReaderDaemon struct {
	sync.RWMutex
	*Logger
	synchronous    bool
	reader         chan interface{}
	readerCallback func(interface{}) error
	errCallback    func(error)
	stopped        bool
}

func NewReaderDaemon(synchronous bool, bufsize uint, readerCallback func(interface{}) error) *ReaderDaemon {
	return &ReaderDaemon{
		Logger:         NewLogger(log),
		synchronous:    synchronous,
		reader:         make(chan interface{}, int(bufsize)),
		readerCallback: readerCallback,
		stopped:        true,
	}
}

func (d *ReaderDaemon) Write(v interface{}) bool {
	go func() {
		d.reader <- v
	}()

	return true
}

func (d *ReaderDaemon) SetErrCallback(errCallback func(error)) *ReaderDaemon {
	d.Lock()
	defer d.Unlock()

	d.errCallback = errCallback

	return d
}

func (d *ReaderDaemon) Start() error {
	_ = d.Stop() // nolint

	if d.readerCallback != nil {
		go d.loop()
	}

	d.Lock()
	defer d.Unlock()

	d.stopped = false
	d.Log().Debug("started")

	return nil
}

func (d *ReaderDaemon) Stop() error {
	if d.IsStopped() {
		return nil
	}

	d.Lock()
	d.stopped = true
	d.Unlock()

	d.Log().Debug("stopped")

	return nil
}

func (d *ReaderDaemon) IsStopped() bool {
	d.RLock()
	defer d.RUnlock()

	return d.stopped
}

func (d *ReaderDaemon) Reader() <-chan interface{} {
	return d.reader
}

func (d *ReaderDaemon) loop() {
	for v := range d.reader {
		if d.synchronous {
			d.runCallback(v)
		} else {
			go d.runCallback(v)
		}
	}
}

func (d *ReaderDaemon) runCallback(v interface{}) {
	err := d.readerCallback(v)
	if err != nil {
		d.Log().Error("error occurred", "error", err)
	}
	if err != nil && d.errCallback != nil {
		go d.errCallback(err)
	}
}
