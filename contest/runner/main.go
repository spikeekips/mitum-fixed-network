package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"

	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/util/logging"
)

var Version string

var (
	log       logging.Logger
	exitHooks []func()
)

type mainFlags struct {
	*contestlib.LogFlags
	Design string `arg name:"node design file" help:"node design file" type:"existingfile"`
}

func main() {
	flags := &mainFlags{
		LogFlags: &contestlib.LogFlags{},
	}
	ctx := kong.Parse(
		flags,
		kong.Name("contest node"),
		kong.Description("contest node"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			NoAppSummary: false,
			Compact:      true,
			Summary:      true,
			Tree:         true,
		}),
		kong.Vars{
			"log":        "",
			"log_level":  "debug",
			"log_format": "terminal",
			"verbose":    "false",
		},
	)

	if l, err := contestlib.SetupLogging(flags.LogFlags, exitHooks); err != nil {
		ctx.FatalIfErrorf(err)
	} else {
		log = l
	}

	connectSignal()

	log.Info().Msg("contest node started")
	log.Debug().Interface("flags", flags).Msg("flags parsed")

	var design *contestlib.NodeDesign
	if d, err := contestlib.LoadDesignFromFile(flags.Design); err != nil {
		log.Error().Err(err).Msg("failed to load design file")

		os.Exit(1)
	} else if err := d.IsValid(nil); err != nil {
		log.Error().Err(err).Msg("invalid design file")

		os.Exit(1)
	} else {
		design = d
		log.Debug().Interface("design", d).Msg("design loaded")
	}

	nr := contestlib.NewNodeRunnerFromDesign(design)
	_ = nr.SetLogger(log)

	if err := nr.Initialize(); err != nil {
		log.Error().Err(err).Msg("failed to generate node from design")

		os.Exit(1)
	}
	log.Debug().Msg("NodeRunner generated")

	fmt.Println(">>>>>>>")
	if gg, err := isaac.NewGenesisBlockV0Generator(nr.Localstate(), nil); err != nil {
		log.Error().Err(err).Msg("failed to create genesis block generator")

		os.Exit(1)
	} else if blk, err := gg.Generate(); err != nil {
		log.Error().Err(err).Msg("failed to generate genesis block")

		os.Exit(1)
	} else {
		fmt.Println("block", blk.Height(), blk.Hash())
	}
	fmt.Println("<<<<<<")

	if err := nr.Start(); err != nil {
		log.Error().Err(err).Msg("failed to start")

		os.Exit(1)
	}

	select {}
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
