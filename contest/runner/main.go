package main

import (
	"io"
	"os"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"

	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/util/logging"
)

var Version string

type mainFlags struct {
	*contestlib.LogFlags
	EventLog string `help:"event log file (default: ${event_log})" default:"${event_log}"`
	Design   string `arg name:"node design file" help:"node design file" type:"existingfile"`
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
			"log":        "", // NOTE if empty, os.Stdout will be used.
			"log_level":  "debug",
			"log_format": "terminal",
			"verbose":    "false",
			"log_color":  "false",
			"event_log":  "",
		},
	)

	var log logging.Logger
	var exitHooks []func()
	if l, err := setupLogging(flags, &exitHooks); err != nil {
		ctx.FatalIfErrorf(err)
	} else {
		log = l
	}

	log.Info().Msg("contest node started")
	log.Debug().Interface("flags", flags).Msg("flags parsed")

	contestlib.ConnectSignal(&exitHooks, log)

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

	if gg, err := isaac.NewGenesisBlockV0Generator(nr.Localstate(), nil); err != nil {
		log.Error().Err(err).Msg("failed to create genesis block generator")

		os.Exit(1)
	} else if blk, err := gg.Generate(); err != nil {
		log.Error().Err(err).Msg("failed to generate genesis block")

		os.Exit(1)
	} else {
		log.Info().Interface("block", blk).Msg("genesis block created")
	}

	if err := nr.Start(); err != nil {
		log.Error().Err(err).Msg("failed to start")

		os.Exit(1)
	}

	select {}
}

func setupLogging(
	flags *mainFlags,
	exitHooks *[]func(),
) (logging.Logger, error) {
	var eventOutput, consoleOutput io.Writer
	if o, err := contestlib.SetupLoggingOutput(flags.Log, flags.LogFormat, flags.LogColor, exitHooks); err != nil {
		return logging.Logger{}, err
	} else {
		consoleOutput = contestlib.NewConsoleWriter(o, zerolog.Level(flags.LogLevel))
	}

	if o, err := contestlib.SetupLoggingOutput(flags.EventLog, "json", false, exitHooks); err != nil {
		return logging.Logger{}, err
	} else {
		eventOutput = o
	}

	output := zerolog.MultiLevelWriter(eventOutput, consoleOutput)
	if l, err := contestlib.SetupLogging(output, flags.LogFlags); err != nil {
		return logging.Logger{}, err
	} else {
		return l, nil
	}
}
