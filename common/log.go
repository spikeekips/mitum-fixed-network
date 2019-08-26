package common

import (
	"time"

	"github.com/rs/zerolog"
)

var NilLog zerolog.Logger = zerolog.Nop()

func init() {
	zerolog.TimestampFieldName = "t"
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.DisableSampling(true)
}

type Logger struct {
	root        zerolog.Logger
	nop         *zerolog.Logger
	l           *zerolog.Logger
	contextFunc []func(zerolog.Context) zerolog.Context
}

func NewLogger(cf func(zerolog.Context) zerolog.Context) *Logger {
	zl := &Logger{nop: &NilLog}
	if cf != nil {
		zl.contextFunc = append(zl.contextFunc, cf)
	}

	return zl
}

func (zl *Logger) SetLogger(l zerolog.Logger) *Logger {
	zl.root = l
	if len(zl.contextFunc) > 0 {
		for _, cf := range zl.contextFunc {
			l = cf(l.With()).Logger()
		}
		zl.l = &l
	} else {
		zl.l = &l
	}

	return zl
}

func (zl *Logger) AddLoggerContext(cf func(zerolog.Context) zerolog.Context) *Logger {
	zl.contextFunc = append(zl.contextFunc, cf)
	if zl.l != nil {
		l := cf(zl.l.With()).Logger()
		zl.l = &l
	}

	return zl
}

func (zl *Logger) RootLog() zerolog.Logger {
	return zl.root
}

func (zl *Logger) Log() *zerolog.Logger {
	if zl.l == nil {
		return zl.nop
	}

	return zl.l
}
