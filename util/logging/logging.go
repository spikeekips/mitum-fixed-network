package logging

import (
	"github.com/rs/zerolog"
)

var (
	NilLog    = zerolog.Nop()
	NilLogger = Logger{Logger: &NilLog}
)

type LogHintedMarshaler interface {
	MarshalLog(string /*key */, Emitter, bool /* is verbose? */) Emitter
}

type HasLogger interface {
	Log() Logger
}

type SetLogger interface {
	SetLogger(Logger) Logger
}

type Logging struct {
	logger       Logger
	contextFuncs []func(Context) Emitter
}

func NewLogging(f func(Context) Emitter) *Logging {
	var fs []func(Context) Emitter
	if f != nil {
		fs = append(fs, f)
	}

	return &Logging{
		logger:       Logger{Logger: &NilLog},
		contextFuncs: fs,
	}
}

func (l *Logging) Log() Logger {
	return l.logger
}

func (l *Logging) SetLogger(nl Logger) Logger {
	if nl.IsNilLogger() {
		return l.logger
	}

	logger := nl.Clone()
	for _, f := range l.contextFuncs {
		f := f
		logger.UpdateContext(func(c zerolog.Context) zerolog.Context {
			return f(newContext(c, logger.IsVerbose())).(Context).Context
		})
	}

	l.logger = logger

	return l.logger
}

type Logger struct {
	*zerolog.Logger
	orig    *zerolog.Logger
	verbose bool
}

func NewLogger(l *zerolog.Logger, verbose bool) Logger {
	n := l.With().Logger()
	return Logger{Logger: &n, orig: l, verbose: verbose}
}

func (l Logger) Level() zerolog.Level {
	return l.Logger.GetLevel()
}

func (l Logger) IsVerbose() bool {
	return l.verbose
}

func (l Logger) Clone() Logger {
	return NewLogger(l.orig, l.verbose)
}

func (l Logger) IsNilLogger() bool {
	if l.Logger == nil {
		return true
	}

	return l.Logger.GetLevel() == zerolog.Disabled
}

func (l Logger) WithLogger(f func(Context) Emitter) Logger {
	n := f(newContext(l.With(), l.verbose)).(Context).Logger()
	return Logger{Logger: &n, orig: l.orig, verbose: l.verbose}
}

func (l Logger) Verbose() *Event {
	if !l.verbose {
		return newEvent(NilLog.Debug())
	}

	nl := l.Logger.With().Bool("verbose", l.verbose).Logger()

	return newEvent(nl.Debug())
}

func (l Logger) VerboseFunc(f func(*Event) Emitter) *Event {
	if !l.verbose {
		return l.Debug()
	}

	return f(l.Verbose()).(*Event)
}

func (l Logger) Panic() *Event {
	return newEvent(l.Logger.Panic())
}

func (l Logger) Fatal() *Event {
	return newEvent(l.Logger.Fatal())
}

func (l Logger) Error() *Event {
	return newEvent(l.Logger.Error())
}

func (l Logger) Warn() *Event {
	return newEvent(l.Logger.Warn())
}

func (l Logger) Info() *Event {
	return newEvent(l.Logger.Info())
}

func (l Logger) Debug() *Event {
	return newEvent(l.Logger.Debug())
}

func (l Logger) Trace() *Event {
	return newEvent(l.Logger.Trace())
}
