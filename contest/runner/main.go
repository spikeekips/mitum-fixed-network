package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
	_ "go.uber.org/automaxprocs"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
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
	"mem_prof_file":   "/mem.prof",
	"trace_prof_file": "/trace.prof",
	"cpu_prof_file":   "/cpu.prof",
	"exit_after":      "0",
}

type mainFlags struct {
	Run     RunCommand  `cmd:"" help:"run contest node runner"`
	Init    InitCommand `cmd:"" help:"initialize"`
	Version struct{}    `cmd:"" help:"print version"`
	Log     []string    `help:"log file"`
}

func main() {
	flags := &mainFlags{
		Run: RunCommand{PprofFlags: &contestlib.PprofFlags{}},
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

	log.Info().Str("version", Version).Msg("contest node started")
	log.Debug().Interface("flags", flags).Msg("flags parsed")

	// check version
	ctx.FatalIfErrorf(func() error {
		return util.Version(Version).IsValid(nil)
	}())

	contestlib.ConnectSignal()

	if ctx.Command() == "version" {
		_, _ = fmt.Fprintln(os.Stdout, Version)

		os.Exit(0)
	}

	ctx.FatalIfErrorf(func() error {
		defer contestlib.ExitHooks.Run()

		return ctx.Run(log)
	}())

	os.Exit(0)
}

type RunCommand struct {
	*contestlib.PprofFlags
	Design    string        `arg:"" name:"node design file" help:"node design file" type:"existingfile"`
	ExitAfter time.Duration `help:"exit after the given duration (default: ${exit_after})" default:"${exit_after}"`
}

func (cmd *RunCommand) Run(log logging.Logger) error {
	if cancel, err := contestlib.RunPprof(cmd.PprofFlags); err != nil {
		return err
	} else {
		contestlib.ExitHooks.Add(func() {
			if err := cancel(); err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err.Error())
			}
		})
	}

	var nr *contestlib.NodeRunner
	if n, err := createNodeRunnerFromDesign(cmd.Design, util.Version(Version), log); err != nil {
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

	if cmd.ExitAfter != 0 {
		<-time.After(cmd.ExitAfter)

		return nil
	}

	select {}
}

type InitCommand struct {
	Design string `arg:"" name:"node design file" help:"node design file" type:"existingfile"`
	Force  bool   `help:"clean the existing environment"`
}

func (cmd *InitCommand) Run(log logging.Logger) error {
	log.Info().Msg("trying to initialize")

	return cmd.run(log)
}

func (cmd *InitCommand) run(log logging.Logger) error {
	var nr *contestlib.NodeRunner
	if n, err := createNodeRunnerFromDesign(cmd.Design, util.Version(Version), log); err != nil {
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

	log.Debug().Msg("checking existing blocks")

	if err := cmd.checkExisting(nr, log); err != nil {
		return err
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

func (cmd *InitCommand) checkExisting(nr *contestlib.NodeRunner, log logging.Logger) error {
	log.Debug().Msg("checking existing blocks")

	var manifest block.Manifest
	if m, found, err := nr.Storage().LastManifest(); err != nil {
		return err
	} else if found {
		manifest = m
	}

	if manifest == nil {
		log.Debug().Msg("not found existing blocks")
	} else {
		log.Debug().Msgf("found existing blocks: block=%d", manifest.Height())

		if !cmd.Force {
			return xerrors.Errorf("environment already exists: block=%d", manifest.Height())
		}

		if err := nr.Storage().Clean(); err != nil {
			return err
		}
		log.Debug().Msg("existing environment cleaned")
	}

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

func createNodeRunnerFromDesign(f string, version util.Version, log logging.Logger) (*contestlib.NodeRunner, error) {
	var encs *encoder.Encoders
	if e, err := encoder.LoadEncoders(
		[]encoder.Encoder{jsonencoder.NewEncoder(), bsonencoder.NewEncoder()},
		contestlib.Hinters...,
	); err != nil {
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
	if n, err := contestlib.NewNodeRunnerFromDesign(design, encs, version); err != nil {
		return nil, xerrors.Errorf("failed to create node runner: %w", err)
	} else {
		nr = n
	}

	_ = nr.SetLogger(log)

	return nr, nil
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
