package main

import (
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/util/encoder"
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
	"log":           "", // NOTE if empty, os.Stdout will be used.
	"log_level":     "debug",
	"log_format":    "terminal",
	"verbose":       "false",
	"log_color":     "false",
	"run_event_log": "",
}

type mainFlags struct {
	Run     RunCommand `cmd:"" help:"run contest node runner"`
	Version struct{}   `cmd:"" help:"print version"`
	*contestlib.LogFlags
}

func main() {
	flags := &mainFlags{
		LogFlags: &contestlib.LogFlags{},
	}
	ctx := kong.Parse(
		flags,
		kong.Name(os.Args[0]),
		kong.Description("contest node runner"),
		kong.UsageOnError(),
		kong.ConfigureHelp(mainHelpOptions),
		mainDefaultVars,
	)

	exitHooks := contestlib.NewExitHooks()
	defer contestlib.RunExitHooks(exitHooks)

	var log logging.Logger
	if o, err := contestlib.SetupLoggingOutput(flags.Log, flags.LogFormat, flags.LogColor, exitHooks); err != nil {
		ctx.FatalIfErrorf(err)
	} else if l, err := contestlib.SetupLogging(o, flags.LogFlags); err != nil {
		ctx.FatalIfErrorf(err)
	} else {
		log = l
	}

	contestlib.ConnectSignal(exitHooks, log)

	if ctx.Command() == "version" {
		_, _ = fmt.Fprintln(os.Stdout, Version)

		os.Exit(0)
	}

	ctx.FatalIfErrorf(ctx.Run(flags, exitHooks))

	os.Exit(0)
}

type RunCommand struct {
	EventLog string `help:"event log file (default: ${run_event_log})" default:"${run_event_log}"`
	Design   string `arg:"" name:"node design file" help:"node design file" type:"existingfile"`
}

func (cmd *RunCommand) Run(flags *mainFlags, exitHooks *[]func()) error {
	var log logging.Logger
	if l, err := setupLogging(flags, cmd.EventLog, exitHooks); err != nil {
		return err
	} else {
		log = l
	}

	log.Info().Msg("contest node started")
	log.Debug().Interface("flags", flags).Msg("flags parsed")

	var nr *contestlib.NodeRunner
	if n, err := loadNodeRunner(cmd.Design); err != nil {
		return xerrors.Errorf("failed to create node runner: %w", err)
	} else {
		nr = n

		_ = nr.SetLogger(log)
	}

	if err := nr.Initialize(); err != nil {
		return xerrors.Errorf("failed to generate node from design: %w", err)
	}

	if err := nr.Start(); err != nil {
		return xerrors.Errorf("failed to start: %w", err)
	}

	select {}
}

func setupLogging(flags *mainFlags, eventLog string, exitHooks *[]func()) (logging.Logger, error) {
	var eventOutput, consoleOutput io.Writer
	if o, err := contestlib.SetupLoggingOutput(flags.Log, flags.LogFormat, flags.LogColor, exitHooks); err != nil {
		return logging.Logger{}, err
	} else {
		consoleOutput = contestlib.NewConsoleWriter(o, zerolog.Level(flags.LogLevel))
	}

	if o, err := contestlib.SetupLoggingOutput(eventLog, "json", false, exitHooks); err != nil {
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
	if d, err := contestlib.LoadNodeDesignFromFile(f, encs); err != nil {
		return nil, xerrors.Errorf("failed to load design file: %w", err)
	} else if err := d.IsValid(nil); err != nil {
		return nil, xerrors.Errorf("invalid design file: %w", err)
	} else {
		return d, nil
	}
}

func loadNodeRunner(f string) (*contestlib.NodeRunner, error) {
	var encs *encoder.Encoders
	if e, err := contestlib.LoadEncoder(); err != nil {
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
