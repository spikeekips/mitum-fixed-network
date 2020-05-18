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
	"log":             "",
	"log_level":       "info",
	"log_format":      "terminal",
	"log_color":       "false",
	"verbose":         "false",
	"start_image":     "golang:latest",
	"start_not_clean": "false",
}

type mainFlags struct {
	Start   cmds.StartCommand   `cmd:"" help:"start contest"`
	Clean   cmds.CleanCommand   `cmd:"" help:"clean contest"`
	Version cmds.VersionCommand `cmd:"" help:"Print version"`
	*contestlib.LogFlags
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

	exitHooks := contestlib.NewExitHooks()

	var log logging.Logger
	if o, err := contestlib.SetupLoggingOutput(flags.Log, flags.LogFormat, flags.LogColor, exitHooks); err != nil {
		ctx.FatalIfErrorf(err)
	} else if l, err := contestlib.SetupLogging(o, flags.LogFlags); err != nil {
		ctx.FatalIfErrorf(err)
	} else {
		log = l
	}

	contestlib.ConnectSignal(exitHooks, log)

	log.Info().Msg("contest started")
	log.Debug().Interface("flags", flags).Msg("flags parsed")

	if err := run(ctx, log, exitHooks); err != nil {
		ctx.FatalIfErrorf(err)
	}

	log.Info().Msg("contest finished")
}

func run(ctx *kong.Context, log logging.Logger, exitHooks *[]func()) error {
	defer contestlib.RunExitHooks(exitHooks)

	return ctx.Run(log, exitHooks)
}
