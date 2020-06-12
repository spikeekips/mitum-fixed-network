package contestlib

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"

	"github.com/spikeekips/mitum/util/logging"
)

func init() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.LevelFieldName = "l"
	zerolog.TimestampFieldName = "t"
	zerolog.MessageFieldName = "m"

	zerolog.DisableSampling(true)
}

type LogFlags struct {
	Log       string    `help:"log file (default: ${log})" default:"${log}"`
	Verbose   bool      `help:"verbose log output (default: false)" default:"${verbose}"`
	LogColor  bool      `help:"show color log" default:"${log_color}"`
	LogLevel  LogLevel  `help:"log level {debug error warn info crit} (default: ${log_level})" default:"${log_level}"`
	LogFormat LogFormat `help:"log format {json terminal} (default: ${log_format})" default:"${log_format}"`
}

func SetupLoggingOutput(f string, format LogFormat, forceColor bool) (io.Writer, error) {
	logFile := strings.TrimSpace(f)

	var output io.Writer
	if len(logFile) < 1 {
		o := os.Stdout
		ExitHooks.Add(func() {
			_ = o.Sync()
		})

		output = o
	} else {
		var outputFile *os.File
		if f, err := os.OpenFile(logFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644); err != nil { // nolint
			return nil, err
		} else {
			outputFile = f
		}

		output = diode.NewWriter(
			outputFile,
			1000,
			0,
			func(missed int) {
				fmt.Fprintf(os.Stderr, "zerolog: dropped %d log mesages", missed)
			},
		)

		if l, ok := output.(diode.Writer); ok {
			ExitHooks.Add(func() {
				_ = l.Close()
			})
		}
	}

	if format == "terminal" {
		var useColor bool
		if forceColor {
			useColor = true
		} else if isatty.IsTerminal(os.Stdout.Fd()) {
			useColor = true
		}

		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339Nano,
			NoColor:    !useColor,
		}
	}

	return output, nil
}

func SetupLogging(output io.Writer, level zerolog.Level, verbose bool) (logging.Logger, error) {
	lc := zerolog.New(output).With().Timestamp()

	if verbose {
		level = zerolog.TraceLevel
	}

	if level <= zerolog.DebugLevel {
		lc = lc.Caller().Stack()
	}

	l := lc.Logger().Level(level)

	return logging.NewLogger(&l, verbose), nil
}

type ConsoleWriter struct {
	w io.Writer
	l zerolog.Level
}

func NewConsoleWriter(w io.Writer, level zerolog.Level) ConsoleWriter {
	return ConsoleWriter{w: w, l: level}
}

func (wr ConsoleWriter) WriteLevel(level zerolog.Level, b []byte) (int, error) {
	if level < wr.l {
		return len(b), nil
	}

	return wr.w.Write(b)
}

func (wr ConsoleWriter) Write(b []byte) (int, error) {
	return wr.w.Write(b)
}
