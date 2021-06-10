package cmds

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"
	"go.uber.org/automaxprocs/maxprocs"

	"github.com/spikeekips/mitum/launch"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
)

var defaultProcesses = []pm.Process{
	process.ProcessorTimeSyncer,
	process.ProcessorEncoders,
	process.ProcessorDatabase,
	process.ProcessorBlockData,
	process.ProcessorLocalNode,
	process.ProcessorProposalProcessor,
	process.ProcessorSuffrage,
	process.ProcessorConsensusStates,
	process.ProcessorNetwork,
	process.ProcessorStartNetwork,
}

var defaultHooks = []pm.Hook{
	pm.NewHook(pm.HookPrefixPost, process.ProcessNameEncoders,
		process.HookNameAddHinters, process.HookAddHinters(launch.EncoderTypes, launch.EncoderHinters)),
	pm.NewHook(pm.HookPrefixPost, process.ProcessNameNetwork,
		process.HookNameSetNetworkHandlers, process.HookSetNetworkHandlers),
	pm.NewHook(pm.HookPrefixPost, process.ProcessNameNetwork,
		process.HookNameNetworkRateLimit, process.HookNetworkRateLimit),
	pm.NewHook(pm.HookPrefixPost, process.ProcessNameLocalNode, process.HookNameSetPolicy, process.HookSetPolicy),
	pm.NewHook(pm.HookPrefixPost, process.ProcessNameLocalNode, process.HookNameNodepool, process.HookNodepool),
	pm.NewHook(pm.HookPrefixPre, process.ProcessNameBlockData,
		process.HookNameCheckBlockDataPath, process.HookCheckBlockDataPath),
}

func DefaultProcesses() *pm.Processes {
	ps := pm.NewProcesses()

	if err := process.Config(ps); err != nil {
		panic(err)
	}

	for i := range defaultProcesses {
		if err := ps.AddProcess(defaultProcesses[i], false); err != nil {
			panic(err)
		}
	}

	for i := range defaultHooks {
		hook := defaultHooks[i]
		if err := ps.AddHook(hook.Prefix, hook.Process, hook.Name, hook.F, true); err != nil {
			panic(err)
		}
	}

	return ps
}

var (
	DefaultName        = "mitum"
	DefaultDescription = "mitum"
	MainOptions        = kong.HelpOptions{NoAppSummary: false, Compact: true, Summary: false, Tree: true}
)

var defaultKongOptions = []kong.Option{
	kong.Name(DefaultName),
	// kong.Description(DefaultDescription),
	kong.UsageOnError(),
	kong.ConfigureHelp(MainOptions),
	LogVars,
	DefaultConfigVars,
	PprofVars,
	NodeConnectVars,
}

func Context(args []string, flags interface{}, options ...kong.Option) (*kong.Context, error) {
	ops := make([]kong.Option, len(defaultKongOptions)+len(options))
	copy(ops, defaultKongOptions)
	copy(ops[len(defaultKongOptions):], options)

	p, err := kong.New(flags, ops...)
	if err != nil {
		return nil, err
	}
	return p.Parse(args)
}

type BaseCommand struct {
	*logging.Logging
	*LogFlags
	*PprofFlags
	LogOutput io.Writer `kong:"-"`
	version   util.Version
	encs      *encoder.Encoders
	jsonenc   *jsonenc.Encoder
	bsonenc   *bsonenc.Encoder
	exithooks []func() error
}

func NewBaseCommand(name string) *BaseCommand {
	return &BaseCommand{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", fmt.Sprintf("command-%s", name))
		}),
		LogFlags:   &LogFlags{},
		PprofFlags: &PprofFlags{},
	}
}

func (cmd *BaseCommand) Initialize(flags interface{}, version util.Version) error {
	if cmd.LogOutput == nil {
		cmd.LogOutput = os.Stdout
	}

	i, err := SetupLoggingFromFlags(cmd.LogFlags, cmd.LogOutput)
	if err != nil {
		return err
	}
	_ = cmd.SetLogger(i)

	_, _ = maxprocs.Set(maxprocs.Logger(func(f string, s ...interface{}) {
		cmd.Log().Debug().Msgf(f, s...)
	}))

	hook, err := RunPprofs(cmd.PprofFlags)
	if err != nil {
		return err
	}
	cmd.exithooks = append(cmd.exithooks, hook)

	cmd.Log().Debug().Interface("flags", flags).Msg("flags parsed")

	if err := version.IsValid(nil); err != nil {
		return err
	}
	cmd.version = version

	return nil
}

func (cmd *BaseCommand) Done() {
	for i := range cmd.exithooks {
		if err := cmd.exithooks[i](); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error: %+v\n", err)
		}
	}

	cmd.Log().Info().Msg("stopped")
}

func (cmd *BaseCommand) Version() util.Version {
	return cmd.version
}

func (cmd *BaseCommand) Encoders() *encoder.Encoders {
	return cmd.encs
}

func (cmd *BaseCommand) JSONEncoder() *jsonenc.Encoder {
	return cmd.jsonenc
}

func (cmd *BaseCommand) BSONEncoder() *bsonenc.Encoder {
	return cmd.bsonenc
}

func (cmd *BaseCommand) LoadEncoders(types []hint.Type, hinters []hint.Hinter) (*encoder.Encoders, error) {
	if cmd.encs != nil {
		return cmd.encs, nil
	}

	if len(hinters) < 1 {
		hinters = launch.EncoderHinters
	}

	ps := pm.NewProcesses().SetContext(context.Background())

	if err := ps.AddProcess(process.ProcessorEncoders, false); err != nil {
		return nil, err
	}

	if err := ps.AddHook(
		pm.HookPrefixPost,
		process.ProcessNameEncoders,
		process.HookNameAddHinters,
		process.HookAddHinters(types, hinters),
		true,
	); err != nil {
		return nil, err
	}

	_ = ps.SetLogger(cmd.Log())

	if err := ps.Run(); err != nil {
		return nil, err
	}

	cmd.encs = new(encoder.Encoders)
	if err := config.LoadEncodersContextValue(ps.Context(), &cmd.encs); err != nil {
		return nil, err
	}

	cmd.jsonenc = new(jsonenc.Encoder)
	if err := config.LoadJSONEncoderContextValue(ps.Context(), &cmd.jsonenc); err != nil {
		return nil, err
	}

	cmd.bsonenc = new(bsonenc.Encoder)
	if err := config.LoadBSONEncoderContextValue(ps.Context(), &cmd.bsonenc); err != nil {
		return nil, err
	}

	return cmd.encs, nil
}
