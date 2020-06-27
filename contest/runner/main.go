package main

import (
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"

	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/contest/runner/cmds"
	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

var Version string = "v0.1-proto3"

var mainHelpOptions = kong.HelpOptions{
	NoAppSummary: false,
	Compact:      false,
	Summary:      true,
	Tree:         true,
}

var mainDefaultVars = kong.Vars{
	"enable_pprofiling": "true",
	"mem_prof_file":     "/mem.prof",
	"trace_prof_file":   "/trace.prof",
	"cpu_prof_file":     "/cpu.prof",
	"exit_after":        "0",
}

type mainFlags struct {
	Run     cmds.RunCommand  `cmd:"" help:"run contest node runner"`
	Init    cmds.InitCommand `cmd:"" help:"initialize"`
	Node    cmds.NodeCommand `cmd:"" help:"run node commands"`
	Version struct{}         `cmd:"" help:"print version"`
	Log     []string         `help:"log file"`
}

func main() {
	flags := &mainFlags{
		Run: cmds.RunCommand{PprofFlags: &launcher.PprofFlags{}},
	}
	ctx := kong.Parse(
		flags,
		kong.Name(os.Args[0]),
		kong.Description("contest node runner"),
		kong.UsageOnError(),
		kong.ConfigureHelp(mainHelpOptions),
		mainDefaultVars,
	)

	var log logging.Logger
	if l, err := setupLogging(flags.Log); err != nil {
		ctx.FatalIfErrorf(err)
	} else {
		log = l
	}

	log.Debug().Interface("flags", flags).Msg("flags parsed")

	version := util.Version(Version)
	ctx.FatalIfErrorf(func() error {
		// NOTE check version
		return version.IsValid(nil)
	}())

	contestlib.ConnectSignal()

	if ctx.Command() == "version" {
		_, _ = fmt.Fprintln(os.Stdout, Version)

		os.Exit(0)
	}

	ctx.FatalIfErrorf(func() error {
		defer contestlib.ExitHooks.Run()

		return ctx.Run(log, version)
	}())

	os.Exit(0)
}

func setupLogging(logs []string) (logging.Logger, error) {
	if len(logs) < 1 {
		logs = []string{""}
	}

	var outputs []io.Writer
	for _, l := range logs {
		if o, err := contestlib.SetupLoggingOutput(l, "json", false); err != nil {
			return logging.Logger{}, err
		} else {
			outputs = append(outputs, o)
		}
	}

	return contestlib.SetupLogging(
		zerolog.MultiLevelWriter(outputs...),
		zerolog.DebugLevel,
		true,
	)
}
