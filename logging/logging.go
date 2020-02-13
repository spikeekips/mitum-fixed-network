package logging

import "github.com/rs/zerolog"

// NilLog is nil logger.
var NilLog *zerolog.Logger

func init() {
	n := zerolog.Nop()
	NilLog = &n
}

type SetLogger interface {
	SetLogger(zerolog.Logger) *Logger
}

// Logger provides message logging with github.com/rs/zerolog. Logger can be
// embedded into struct.
type Logger struct {
	l           *zerolog.Logger
	contextFunc []func(zerolog.Context) zerolog.Context
}

// NewLogger creates new Logger.
func NewLogger(cf func(zerolog.Context) zerolog.Context) *Logger {
	zl := &Logger{l: NilLog}
	if cf != nil {
		zl.contextFunc = append(zl.contextFunc, cf)
	}

	return zl
}

// SetLogger does not support asynchronous access, so it must be called at a
// created time.
func (zl *Logger) SetLogger(l zerolog.Logger) *Logger {
	if len(zl.contextFunc) > 0 {
		for _, cf := range zl.contextFunc {
			l = cf(l.With()).Logger()
		}
	}

	zl.l = &l

	return zl
}

// NewLogger creates new Logger with existing contexts from Logger.
func (zl *Logger) NewLogger(cf func(zerolog.Context) zerolog.Context) *Logger {
	contextFunc := zl.contextFunc
	contextFunc = append(contextFunc, cf)

	var l zerolog.Logger
	if zl.l != NilLog {
		l = *zl.l
		if cf != nil {
			l = cf(l.With()).Logger()
		}

		for _, f := range zl.contextFunc {
			l = f(l.With()).Logger()
		}
	}

	return &Logger{
		l:           &l,
		contextFunc: contextFunc,
	}
}

// Log returns Logger.
func (zl *Logger) Log() *zerolog.Logger {
	if zl.l == nil {
		return NilLog
	}

	return zl.l
}
