package main

import (
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/operation"
	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/isaac"
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
	Run     RunCommand  `cmd:"" help:"run contest node runner"`
	Init    InitCommand `cmd:"" help:"initialize"`
	Version struct{}    `cmd:"" help:"print version"`
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

	log.Info().Str("version", Version).Msg("contest node started")
	log.Debug().Interface("flags", flags).Msg("flags parsed")

	var nr *contestlib.NodeRunner
	if n, err := createNodeRunnerFromDesign(cmd.Design, log); err != nil {
		return xerrors.Errorf("failed to create node runner: %w", err)
	} else {
		nr = n
	}

	if err := nr.Initialize(); err != nil {
		return xerrors.Errorf("failed to generate node from design: %w", err)
	}

	if err := nr.Start(); err != nil {
		return xerrors.Errorf("failed to start: %w", err)
	}

	select {}
}

type InitCommand struct {
	Design string `arg:"" name:"node design file" help:"node design file" type:"existingfile"`
	Force  bool   `help:"clean the existing environment"`
}

func (cmd *InitCommand) Run(flags *mainFlags, exitHooks *[]func()) error {
	var log logging.Logger
	if l, err := setupLogging(flags, "", exitHooks); err != nil {
		return err
	} else {
		log = l
	}

	log.Info().Msg("trying to initialize")

	log.Debug().Interface("flags", flags).Msg("flags parsed")

	return cmd.run(log)
}

func (cmd *InitCommand) run(log logging.Logger) error {
	var nr *contestlib.NodeRunner
	if n, err := createNodeRunnerFromDesign(cmd.Design, log); err != nil {
		return err
	} else {
		nr = n
	}

	var ops []operation.Operation
	for _, f := range nr.Design().GenesisOperations {
		if op, err := cmd.loadOperationBody(f.Body(), nr.Design()); err != nil {
			return err
		} else {
			log.Debug().Interface("operation", op).Msg("operation loaded")

			ops = append(ops, op)
		}
	}
	log.Debug().Int("operations", len(ops)).Msg("operations loaded")

	if err := nr.Initialize(); err != nil {
		return xerrors.Errorf("failed to generate node from design: %w", err)
	}

	// check the existing blocks
	log.Debug().Msg("checking existing blocks")
	if blk, err := nr.Storage().LastBlock(); err != nil {
		return err
	} else {
		if blk == nil {
			log.Debug().Msg("not found existing blocks")
		} else {
			log.Debug().Msgf("found existing blocks: block=%d", blk.Height())

			if !cmd.Force {
				return xerrors.Errorf("environment already exists: block=%d", blk.Height())
			}

			if err := nr.Storage().Clean(); err != nil {
				return err
			}
			log.Debug().Msg("existing environment cleaned")
		}
	}

	log.Debug().Msg("trying to create genesis block")
	if gg, err := isaac.NewGenesisBlockV0Generator(nr.Localstate(), ops); err != nil {
		return xerrors.Errorf("failed to create genesis block generator: %w", err)
	} else if blk, err := gg.Generate(); err != nil {
		return xerrors.Errorf("failed to generate genesis block: %w", err)
	} else {
		log.Info().
			Dict("block", logging.Dict().Hinted("height", blk.Height()).Hinted("hash", blk.Hash())).
			Msg("genesis block created")
	}

	log.Info().Msg("genesis block created")
	log.Info().Msg("iniialized")

	return nil
}

func (cmd *InitCommand) loadOperationBody(body interface{}, design *contestlib.NodeDesign) (
	operation.Operation, error,
) {
	switch t := body.(type) {
	case isaac.PolicyOperationBodyV0:
		token := []byte("genesis-policies-from-contest")
		var fact isaac.SetPolicyOperationFactV0
		if f, err := isaac.NewSetPolicyOperationFactV0(design.Privatekey().Publickey(), token, t); err != nil {
			return nil, err
		} else {
			fact = f
		}

		if op, err := isaac.NewSetPolicyOperationV0FromFact(
			design.Privatekey(),
			fact,
			design.NetworkID(),
		); err != nil {
			return nil, xerrors.Errorf("failed to create SetPolicyOperation: %w", err)
		} else {
			return op, nil
		}
	default:
		return nil, xerrors.Errorf("unsupported body for genesis operation: %T", body)
	}
}

func createNodeRunnerFromDesign(f string, log logging.Logger) (*contestlib.NodeRunner, error) {
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

	_ = nr.SetLogger(log)

	return nr, nil
}

func setupLogging(flags *mainFlags, eventLog string, exitHooks *[]func()) (logging.Logger, error) {
	var consoleOutput io.Writer
	if o, err := contestlib.SetupLoggingOutput(flags.Log, flags.LogFormat, flags.LogColor, exitHooks); err != nil {
		return logging.Logger{}, err
	} else {
		consoleOutput = contestlib.NewConsoleWriter(o, zerolog.Level(flags.LogLevel))
	}

	if len(eventLog) < 1 {
		if l, err := contestlib.SetupLogging(consoleOutput, flags.LogFlags); err != nil {
			return logging.Logger{}, err
		} else {
			return l, nil
		}
	}

	var eventOutput io.Writer
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
