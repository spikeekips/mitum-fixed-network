package main

import (
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"

	"github.com/spikeekips/mitum/contest/cmds"
	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/util/logging"
)

var Version string

type mainFlags struct {
	Start   cmds.StartCommand `cmd:"" help:"start contest"`
	Version struct{}          `cmd:"" help:"Print version"`
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
		kong.ConfigureHelp(kong.HelpOptions{
			NoAppSummary: false,
			Compact:      true,
			Summary:      false,
			Tree:         true,
		}),
		kong.Vars{
			"log":        "",
			"log_level":  "debug",
			"log_format": "terminal",
			"verbose":    "false",
			"nodes":      "1",                               // TODO set optional
			"networkID":  fmt.Sprintf("contest-network-id"), // TODO set optional
		},
	)

	var log logging.Logger
	var exitHooks []func()
	var consoleOutput io.Writer
	if o, err := contestlib.SetupLoggingOutput(flags.Log, flags.LogFormat, flags.LogColor, &exitHooks); err != nil {
		ctx.FatalIfErrorf(err)
	} else {
		consoleOutput = o
	}

	if l, err := contestlib.SetupLogging(consoleOutput, flags.LogFlags); err != nil {
		ctx.FatalIfErrorf(err)
	} else {
		log = l
	}

	contestlib.ConnectSignal(&exitHooks, log)

	log.Info().Msg("contest started")
	log.Debug().Interface("flags", flags).Msg("flags parsed")

	if ctx.Command() == "version" {
		_, _ = fmt.Fprintln(os.Stdout, Version)

		os.Exit(0)
	}

	ctx.FatalIfErrorf(ctx.Run())

	log.Info().Msg("contest finished")

	os.Exit(0)
}
