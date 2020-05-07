package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"

	"github.com/spikeekips/mitum/contest/cmds"
	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/util/logging"
)

var Version string

var (
	log       logging.Logger
	exitHooks []func()
)

type mainFlags struct {
	*contestlib.LogFlags
	Version struct{}          `cmd help:"Print version"`
	Start   cmds.StartCommand `cmd help:"start contest"`
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

	if l, err := contestlib.SetupLogging(flags.LogFlags, exitHooks); err != nil {
		ctx.FatalIfErrorf(err)
	} else {
		log = l
	}

	connectSignal()

	log.Info().Msg("contest started")
	log.Debug().Interface("flags", flags).Msg("flags parsed")

	switch ctx.Command() {
	case "version":
		fmt.Fprintln(os.Stdout, Version)

		os.Exit(0)
	}

	ctx.FatalIfErrorf(ctx.Run())

	log.Info().Msg("contest finished")

	os.Exit(0)
}

func connectSignal() {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	go func() {
		s := <-sigc

		for _, h := range exitHooks {
			h()
		}

		log.Fatal().
			Str("sig", s.String()).
			Msg("contest stopped by force")

		os.Exit(1)
	}()
}
