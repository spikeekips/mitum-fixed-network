package util

import (
	"sync"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/util/logging"
)

type FunctionDaemon struct {
	sync.RWMutex
	*logging.Logging
	fn           func(chan struct{}) error
	stoppingChan chan struct{}
	stopChan     chan struct{}
	stoppingWait *sync.WaitGroup
	isDebug      bool
	isBlocked    bool
}

func NewFunctionDaemon(fn func(chan struct{}) error, isDebug bool) *FunctionDaemon {
	dm := &FunctionDaemon{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "functondaemon")
		}),
		fn:       fn,
		stopChan: make(chan struct{}, 2),
		isDebug:  isDebug,
	}

	return dm
}

func (dm *FunctionDaemon) SetBlocking(blocked bool) *FunctionDaemon {
	dm.isBlocked = blocked

	return dm
}

func (dm *FunctionDaemon) IsStarted() bool {
	dm.RLock()
	defer dm.RUnlock()

	return dm.stoppingChan != nil
}

func (dm *FunctionDaemon) IsStopped() bool {
	dm.RLock()
	defer dm.RUnlock()

	return dm.stoppingChan == nil
}

func (dm *FunctionDaemon) Start() error {
	if dm.isDebug {
		dm.Log().Debug().Msg("trying to start")
	}

	if dm.IsStarted() {
		return DaemonAlreadyStartedError
	}

	if dm.isBlocked {
		return dm.fn(dm.stopChan)
	}

	{
		dm.Lock()
		dm.stoppingChan = make(chan struct{}, 2)

		dm.stoppingWait = &sync.WaitGroup{}
		dm.stoppingWait.Add(1)
		dm.Unlock()
	}

	go dm.kill()

	go func() {
		if err := dm.fn(dm.stopChan); err != nil && dm.isDebug {
			dm.Log().Error().Err(err).Msg("occurred in daemon function")
		}
		dm.stoppingChan <- struct{}{}
	}()

	if dm.isDebug {
		dm.Log().Debug().Msg("started")
	}

	return nil
}

func (dm *FunctionDaemon) kill() {
	<-dm.stoppingChan
	dm.stoppingWait.Done()

	dm.Lock()
	dm.stoppingChan = nil
	dm.Unlock()

	if dm.isDebug {
		dm.Log().Debug().Msg("stopped")
	}
}

func (dm *FunctionDaemon) Stop() error {
	if dm.isDebug {
		dm.Log().Debug().Msg("trying to stop")
	}

	if dm.IsStopped() {
		return DaemonAlreadyStoppedError
	}

	dm.stopChan <- struct{}{}
	dm.stoppingWait.Wait()

	if dm.isDebug {
		dm.Log().Debug().Msg("stopped")
	}
	return nil
}
