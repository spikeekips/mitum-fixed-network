package commands

import "github.com/spikeekips/mitum/contest/flag"

type CommonFlags struct {
	Log       *string        `help:"log output directory" default:"${log}" type:"existingdir"`
	LogLevel  flag.LogLevel  `help:"log level {debug error warn info crit}" default:"${log_level}"`
	LogFormat flag.LogFormat `help:"log format {json terminal}" default:"${log_format}"`
}
