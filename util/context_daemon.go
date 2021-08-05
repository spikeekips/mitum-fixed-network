package util

import (
	"context"
	"sync"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/util/logging"
)

type ContextDaemon struct {
	sync.RWMutex
	*logging.Logging
	ctxLock            sync.RWMutex
	callback           func(context.Context) error
	callbackCtx        context.Context
	callbackCancelFunc func()
	ctx                context.Context
	stopfunc           func()
}

func NewContextDaemon(name string, startfunc func(context.Context) error) *ContextDaemon {
	return &ContextDaemon{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "context-daemon").Str("daemon", name)
		}),
		callback: startfunc,
	}
}

func (dm *ContextDaemon) IsStarted() bool {
	dm.ctxLock.RLock()
	defer dm.ctxLock.RUnlock()

	return dm.callbackCancelFunc != nil
}

func (dm *ContextDaemon) Start() error {
	if dm.IsStarted() {
		return DaemonAlreadyStartedError
	}

	_ = dm.Wait(context.Background())

	dm.Log().Debug().Msg("started")

	return nil
}

func (dm *ContextDaemon) StartWithContext(ctx context.Context) error {
	if dm.IsStarted() {
		return DaemonAlreadyStartedError
	}

	_ = dm.Wait(ctx)

	return nil
}

func (dm *ContextDaemon) Wait(ctx context.Context) <-chan error {
	dm.Lock()
	defer dm.Unlock()

	ch := make(chan error, 1)

	if dm.IsStarted() {
		go func() {
			ch <- DaemonAlreadyStartedError
		}()

		return ch
	}

	nctx, _, _, finish := dm.getCtx(ctx)

	go func(nctx context.Context, ch chan error) {
		err := dm.callback(nctx)

		finish()
		dm.releaseCallbackCtx()

		ch <- err
		close(ch)
	}(nctx, ch)

	return ch
}

func (dm *ContextDaemon) Stop() error {
	dm.Lock()
	defer dm.Unlock()

	if !dm.IsStarted() {
		return DaemonAlreadyStoppedError
	}

	dm.callbackCancel()
	dm.waitCallbackFinished()
	dm.releaseCallbackCtx()

	dm.Log().Debug().Msg("stopped")

	return nil
}

func (dm *ContextDaemon) getCtx(ctx context.Context) (context.Context, func(), context.Context, func()) {
	dm.ctxLock.Lock()
	defer dm.ctxLock.Unlock()

	dm.callbackCtx, dm.callbackCancelFunc = context.WithCancel(ctx)
	dm.ctx, dm.stopfunc = context.WithCancel(context.Background())

	return dm.callbackCtx, dm.callbackCancelFunc, dm.ctx, dm.stopfunc
}

func (dm *ContextDaemon) releaseCallbackCtx() {
	dm.ctxLock.Lock()
	defer dm.ctxLock.Unlock()

	if dm.callbackCtx == nil {
		return
	}

	dm.callbackCtx = nil
	dm.callbackCancelFunc = nil
}

func (dm *ContextDaemon) callbackCancel() {
	dm.ctxLock.RLock()
	defer dm.ctxLock.RUnlock()

	dm.callbackCancelFunc()
}

func (dm *ContextDaemon) waitCallbackFinished() {
	dm.ctxLock.RLock()
	defer dm.ctxLock.RUnlock()

	<-dm.ctx.Done()
}
