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

	stopOnce       *sync.Once
	synchronous    bool
	stop           chan struct{}
	reader         chan interface{}
	readerCallback func(interface{}) error
	errCallback    func(error)
}

func NewReaderDaemon(synchronous bool, bufsize uint, readerCallback func(interface{}) error) *ReaderDaemon {
	return &ReaderDaemon{
		Logger:         NewLogger(log),
		synchronous:    synchronous,
		reader:         make(chan interface{}, int(bufsize)),
		readerCallback: readerCallback,
	}
}

func (d *ReaderDaemon) Reader() <-chan interface{} {
	return d.reader
}

func (d *ReaderDaemon) Write(v interface{}) bool {
	if d.IsStopped() {
		return false
	}

	d.reader <- v

	return true
}

func (d *ReaderDaemon) Close() error {
	if err := d.Stop(); err != nil {
		return err
	}

	d.Lock()
	defer d.Unlock()

	close(d.reader)

	return nil
}

func (d *ReaderDaemon) SetErrCallback(errCallback func(error)) *ReaderDaemon {
	d.Lock()
	defer d.Unlock()

	d.errCallback = errCallback

	return d
}

func (d *ReaderDaemon) Start() error {
	if !d.IsStopped() {
		return DaemonAleadyStartedError
	}

	d.Lock()
	defer d.Unlock()

	d.stop = make(chan struct{}, 2)
	d.stopOnce = new(sync.Once)

	if d.readerCallback != nil {
		go d.loop()
	}

	return nil
}

func (d *ReaderDaemon) Stop() error {
	d.stopOnce.Do(func() {
		d.Lock()
		defer d.Unlock()

		d.stop <- struct{}{}
		close(d.stop)
	})

	return nil
}

func (d *ReaderDaemon) IsStopped() bool {
	d.RLock()
	defer d.RUnlock()

	return d.stop == nil
}

func (d *ReaderDaemon) loop() {
end:
	for {
		select {
		case <-d.stop:
			break end
		case v, notClosed := <-d.reader:
			if !notClosed {
				break end
			}

			if d.synchronous {
				d.runCallback(v)
			} else {
				go d.runCallback(v)
			}
		}
	}

	d.Lock()
	defer d.Unlock()

	d.stop = nil
}

func (d *ReaderDaemon) runCallback(v interface{}) {
	if d.readerCallback == nil {
		return
	}

	err := d.readerCallback(v)
	if err != nil {
		d.Log().Error("error occurred", "error", err)
	}
	if err != nil && d.errCallback != nil {
		go d.errCallback(err)
	}
}
