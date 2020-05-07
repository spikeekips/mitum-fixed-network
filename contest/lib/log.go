package contestlib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"github.com/spikeekips/mitum/util/logging"
)

type LogFlags struct {
	Log       string    `help:"log file (default: ${log})" default:"${log}"`
	LogLevel  LogLevel  `help:"log level {debug error warn info crit} (default: ${log_level})" default:"${log_level}"`
	LogFormat LogFormat `help:"log format {json terminal} (default: ${log_format})" default:"${log_format}"`
	Verbose   bool      `help:"verbose log output (default: false)" default:"${verbose}"`
}

func SetupLogging(flags *LogFlags, exitHooks []func()) (logging.Logger, error) {
	zerolog.TimestampFieldName = "t"
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.MessageFieldName = "m"

	zerolog.DisableSampling(true)

	logFile := strings.TrimSpace(flags.Log)

	var output io.Writer
	if len(logFile) < 1 {
		output = os.Stdout
	} else {
		if curdir, err := os.Getwd(); err != nil {
			return logging.Logger{}, err
		} else if f, err := os.OpenFile(
			filepath.Join(curdir, logFile),
			os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644); err != nil {
			return logging.Logger{}, err
		} else {
			output = diode.NewWriter(
				f,
				1000,
				0,
				func(missed int) {
					fmt.Fprintf(os.Stderr, "zerolog: dropped %d log mesages", missed)
				},
			)
		}

		exitHooks = append(exitHooks, func() {
			if l, ok := output.(diode.Writer); ok {
				_ = l.Close()
			}
		})
	}

	if flags.LogFormat == "terminal" {
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339Nano,
		}
	}

	lc := zerolog.New(output).With().Timestamp()

	level := zerolog.Level(flags.LogLevel)
	if level == zerolog.DebugLevel {
		lc = lc.Caller().Stack()
	}

	l := lc.Logger().Level(level)

	return logging.NewLogger(&l, flags.Verbose), nil
}
