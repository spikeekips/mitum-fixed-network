package common

import (
	"fmt"
	"sync"

	"github.com/inconshreveable/log15"
)

type Loggerable interface {
	Log() log15.Logger
	SetLogger(log15.Logger) *Logger
	LogContext() log15.Ctx
	SetLogContext(log15.Ctx, ...interface{}) *Logger
}

type Logger struct {
	sync.RWMutex
	logCtx log15.Ctx
	log    log15.Logger
}

func NewLogger(log log15.Logger, args ...interface{}) *Logger {
	l := &Logger{log: log}
	l.SetLogContext(nil, args...)

	return l
}

func (l *Logger) LogContext() log15.Ctx {
	l.RLock()
	defer l.RUnlock()

	if l.logCtx == nil {
		return log15.Ctx{}
	}

	return l.logCtx
}

func (l *Logger) SetLogContext(ctx log15.Ctx, args ...interface{}) *Logger {
	if len(args)%2 != 0 {
		panic(fmt.Errorf("invalid number of args: %v", len(args)))
	}

	l.Lock()
	defer l.Unlock()

	if l.logCtx == nil {
		l.logCtx = log15.Ctx{}
	}

	if ctx != nil { // merge
		for k, v := range ctx {
			l.logCtx[k] = v
		}
	}

	for i := 0; i < len(args); i += 2 {
		k, ok := args[i].(string)
		if !ok {
			panic(fmt.Errorf("key should be string: %T found", args[i]))
		}
		l.logCtx[k] = args[i+1]
	}

	l.log = l.log.New(l.logCtx)

	return l
}

func (l *Logger) SetLogger(log log15.Logger) *Logger {
	l.Lock()
	defer l.Unlock()

	l.log = log

	if l.logCtx != nil {
		l.log = l.log.New(l.logCtx)
	}

	return l
}

func (l *Logger) Log() log15.Logger {
	l.RLock()
	defer l.RUnlock()

	return l.log
}
