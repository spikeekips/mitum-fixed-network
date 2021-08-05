package logging

import "github.com/rs/zerolog"

type HasLogger interface {
	Log() *zerolog.Logger
}

type SetLogging interface {
	SetLogging(*Logging) *Logging
}

type Logging struct {
	l    zerolog.Logger
	orig zerolog.Logger
	f    func(zerolog.Context) zerolog.Context
}

func NewLogging(f func(zerolog.Context) zerolog.Context) *Logging {
	nop := zerolog.Nop()
	return &Logging{
		l:    nop,
		orig: nop,
		f:    f,
	}
}

func (lg *Logging) Log() *zerolog.Logger {
	return &lg.l
}

func (lg *Logging) SetLogger(l zerolog.Logger) *Logging {
	lg.orig = l
	if lg.f != nil {
		lg.l = lg.f(lg.orig.With()).Logger()
	} else {
		lg.l = l
	}

	return lg
}

func (lg *Logging) SetLogging(l *Logging) *Logging {
	return lg.SetLogger(l.orig)
}

func (lg *Logging) IsTraceLog() bool {
	return lg.l.GetLevel() == zerolog.TraceLevel
}
