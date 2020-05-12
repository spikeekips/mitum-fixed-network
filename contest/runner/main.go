package main

import (
	"io"
	"os"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/util/encoder"
	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
)

var mainHelpOptions = kong.HelpOptions{
	NoAppSummary: false,
	Compact:      true,
	Summary:      true,
	Tree:         true,
}

var mainDefaultVars = kong.Vars{
	"log":        "", // NOTE if empty, os.Stdout will be used.
	"log_level":  "debug",
	"log_format": "terminal",
	"verbose":    "false",
	"log_color":  "false",
	"event_log":  "",
}

type mainFlags struct {
	*contestlib.LogFlags
	EventLog string `help:"event log file (default: ${event_log})" default:"${event_log}"`
	Design   string `arg:"" name:"node design file" help:"node design file" type:"existingfile"`
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
		kong.ConfigureHelp(mainHelpOptions),
		mainDefaultVars,
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

	var nr *contestlib.NodeRunner
	if n, err := loadNodeRunner(flags.Design); err != nil {
		log.Error().Err(err).Msg("failed to create node runner")

		os.Exit(1)
	} else {
		nr = n

		_ = nr.SetLogger(log)
	}

	if err := nr.Initialize(); err != nil {
		log.Error().Err(err).Msg("failed to generate node from design")

		os.Exit(1)
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

func loadDesign(f string, encs *encoder.Encoders) (*contestlib.NodeDesign, error) {
	if d, err := contestlib.LoadDesignFromFile(f, encs); err != nil {
		return nil, xerrors.Errorf("failed to load design file: %w", err)
	} else if err := d.IsValid(nil); err != nil {
		return nil, xerrors.Errorf("invalid design file: %w", err)
	} else {
		return d, nil
	}
}

func loadEncoder() (*encoder.Encoders, error) {
	encs := encoder.NewEncoders()
	{
		enc := jsonencoder.NewEncoder()
		if err := encs.AddEncoder(enc); err != nil {
			return nil, err
		}
	}

	{
		enc := bsonencoder.NewEncoder()
		if err := encs.AddEncoder(enc); err != nil {
			return nil, err
		}
	}

	for i := range contestlib.Hinters {
		hinter, ok := contestlib.Hinters[i][1].(hint.Hinter)
		if !ok {
			return nil, xerrors.Errorf("not hint.Hinter: %T", contestlib.Hinters[i])
		}

		if err := encs.AddHinter(hinter); err != nil {
			return nil, err
		}
	}

	return encs, nil
}

func loadNodeRunner(f string) (*contestlib.NodeRunner, error) {
	var encs *encoder.Encoders
	if e, err := loadEncoder(); err != nil {
		return nil, xerrors.Errorf("failed to load encoders: %w", err)
	} else {
		encs = e
	}

	var design *contestlib.NodeDesign
	if d, err := loadDesign(f, encs); err != nil {
		return nil, xerrors.Errorf("failed to load design: %w", err)
	} else {
		design = d
	}

	var nr *contestlib.NodeRunner
	if n, err := contestlib.NewNodeRunnerFromDesign(design, encs); err != nil {
		return nil, xerrors.Errorf("failed to create node runner: %w", err)
	} else {
		nr = n
	}

	return nr, nil
}
