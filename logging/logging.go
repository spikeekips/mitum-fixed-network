package logging

import (
	"github.com/rs/zerolog"
)

var NilLog zerolog.Logger = zerolog.Nop()

type SetLogger interface {
	SetLogger(Logger) Logger
}

type Logging struct {
	logger       Logger
	contextFuncs []func(zerolog.Context) zerolog.Context
}

func NewLogging(f func(zerolog.Context) zerolog.Context) *Logging {
	var fs []func(zerolog.Context) zerolog.Context
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
		logger.UpdateContext(f)
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
	return Logger{Logger: cp(l), orig: l, verbose: verbose}
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

func (l Logger) Verbose() *zerolog.Event {
	if !l.verbose {
		return NilLog.Debug()
	}

	nl := l.Logger.With().Bool("verbose", l.verbose).Logger()

	return nl.Debug()
}

func (l Logger) VerboseFunc(f func(*zerolog.Event) *zerolog.Event) *zerolog.Event {
	if !l.verbose {
		return NilLog.Debug()
	}

	return f(l.Verbose())
}

func cp(c *zerolog.Logger) *zerolog.Logger {
	n := *c

	return &n
}
