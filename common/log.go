package common

import (
	"time"

	"github.com/rs/zerolog"
)

func init() {
	zerolog.TimestampFieldName = "t"
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.DisableSampling(true)
}

type ZLogger struct {
	root        zerolog.Logger
	nop         *zerolog.Logger
	l           *zerolog.Logger
	contextFunc []func(zerolog.Context) zerolog.Context
}

func NewZLogger(cf func(zerolog.Context) zerolog.Context) *ZLogger {
	n := zerolog.Nop()
	zl := &ZLogger{nop: &n}
	if cf != nil {
		zl.contextFunc = append(zl.contextFunc, cf)
	}

	return zl
}

func (zl *ZLogger) SetLogger(l zerolog.Logger) *ZLogger {
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

func (zl *ZLogger) AddLoggerContext(cf func(zerolog.Context) zerolog.Context) *ZLogger {
	zl.contextFunc = append(zl.contextFunc, cf)
	if zl.l != nil {
		l := cf(zl.l.With()).Logger()
		zl.l = &l
	}

	return zl
}

func (zl *ZLogger) RootLog() zerolog.Logger {
	return zl.root
}

func (zl *ZLogger) Log() *zerolog.Logger {
	if zl.l == nil {
		return zl.nop
	}

	return zl.l
}
