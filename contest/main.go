package main

import (
	"os"

	"github.com/alecthomas/kong"

	"github.com/spikeekips/mitum/contest/cmds"
	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/util/logging"
)

var mainOptions = kong.HelpOptions{NoAppSummary: false, Compact: true, Summary: false, Tree: true}

var mainVars = kong.Vars{
	"log":         "",
	"log_level":   "info",
	"log_format":  "terminal",
	"log_color":   "false",
	"verbose":     "false",
	"start_image": "golang:latest",
	"exit_after":  "5m",
	"alias":       cmds.DefaultAlias,
}

type mainFlags struct {
	*contestlib.LogFlags
	Log     string              `help:"log output"`
	Start   cmds.StartCommand   `cmd:"" help:"start contest"`
	Clean   cmds.CleanCommand   `cmd:"" help:"clean contest"`
	Version cmds.VersionCommand `cmd:"" help:"Print version"`
}

func main() {
	flags := &mainFlags{
		LogFlags: &contestlib.LogFlags{},
	}
	ctx := kong.Parse(
		flags,
		kong.Name("contest"),
		kong.Description("Consensus tester"),
		kong.UsageOnError(),
		kong.ConfigureHelp(mainOptions),
		mainVars,
	)

	if ctx.Command() == "version" {
		if err := ctx.Run(); err != nil {
			ctx.FatalIfErrorf(err)
		}

		os.Exit(0)
	}

	var log logging.Logger
	if o, err := contestlib.SetupLoggingOutput(flags.Log, flags.LogFormat, flags.LogColor, os.Stdout); err != nil {
		ctx.FatalIfErrorf(err)
	} else if l, err := contestlib.SetupLogging(o, flags.LogLevel.Zero(), flags.Verbose); err != nil {
		ctx.FatalIfErrorf(err)
	} else {
		log = l
	}

	contestlib.ConnectSignal()

	log.Info().Msg("contest started")
	log.Debug().Interface("flags", flags).Msg("flags parsed")

	ctx.FatalIfErrorf(run(ctx, log))

	os.Exit(0)
}

func run(ctx *kong.Context, log logging.Logger) error {
	defer log.Info().Msg("contest finished")
	defer contestlib.ExitHooks.Run()

	return ctx.Run(log)
}
