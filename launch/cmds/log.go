package cmds

import (
	"bytes"
	"io"
	"time"

	"github.com/alecthomas/kong"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
)

func init() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.LevelFieldName = "l"
	zerolog.TimestampFieldName = "t"
	zerolog.MessageFieldName = "m"
	zerolog.TimestampFunc = localtime.UTCNow
	zerolog.InterfaceMarshalFunc = util.JSON.Marshal
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	zerolog.DisableSampling(true)
}

var LogVars = kong.Vars{
	"log":        "",
	"log_level":  "info",
	"log_format": "terminal",
	"log_color":  "false",
}

type LogFlags struct {
	LogColor  bool      `help:"show color log" default:"${log_color}"`                                                       // revive:disable-line:struct-tag,line-length-limit
	LogLevel  LogLevel  `help:"log level {trace debug error warn info crit} (default: ${log_level})" default:"${log_level}"` // revive:disable-line:struct-tag,line-length-limit
	LogFormat LogFormat `help:"log format {json terminal} (default: ${log_format})" default:"${log_format}"`
	LogFile   []string  `name:"log" help:"log file"`
}

type LogLevel zerolog.Level

func (ll LogLevel) Zero() zerolog.Level {
	return zerolog.Level(ll)
}

func (ll LogLevel) MarshalText() ([]byte, error) {
	return []byte(zerolog.Level(ll).String()), nil
}

func (ll *LogLevel) UnmarshalText(b []byte) error {
	lvl, err := zerolog.ParseLevel(string(b))
	if err != nil {
		return err
	}

	*ll = LogLevel(lvl)

	return nil
}

type LogFormat string

func (lf *LogFormat) UnmarshalText(b []byte) error {
	s := string(bytes.TrimSpace(bytes.ToLower(b)))
	switch s {
	case "json":
	case "terminal":
	default:
		return errors.Errorf("invalid log_format: %q", s)
	}

	*lf = LogFormat(s)

	return nil
}

func SetupLoggingFromFlags(flags *LogFlags, defaultout io.Writer) (*logging.Logging, error) {
	output := defaultout
	if len(flags.LogFile) > 0 {
		i, err := logging.Outputs(flags.LogFile)
		if err != nil {
			return nil, err
		}
		output = i
	}

	return logging.Setup(
		output,
		zerolog.Level(flags.LogLevel),
		string(flags.LogFormat),
		flags.LogColor,
	), nil
}
