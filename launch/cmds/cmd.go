package cmds

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/alecthomas/kong"
	"go.uber.org/automaxprocs/maxprocs"

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
	BlocksVars,
	DefaultConfigVars,
	PprofVars,
}

func Context(args []string, flags interface{}, options ...kong.Option) (*kong.Context, error) {
	ops := make([]kong.Option, len(defaultKongOptions)+len(options))
	copy(ops, defaultKongOptions)
	copy(ops[len(defaultKongOptions):], options)

	if p, err := kong.New(flags, ops...); err != nil {
		return nil, err
	} else {
		return p.Parse(args)
	}
}

type BaseCommand struct {
	*logging.Logging
	*LogFlags
	*PprofFlags
	version   util.Version
	encs      *encoder.Encoders
	jsonenc   *jsonenc.Encoder
	bsonenc   *bsonenc.Encoder
	exithooks []func() error
	done      sync.Once
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
	if i, err := SetupLoggingFromFlags(cmd.LogFlags); err != nil {
		return err
	} else {
		_ = cmd.SetLogger(i)
	}

	_, _ = maxprocs.Set(maxprocs.Logger(func(f string, s ...interface{}) {
		cmd.Log().Debug().Msgf(f, s...)
	}))

	if hook, err := RunPprofs(cmd.PprofFlags); err != nil {
		return err
	} else {
		cmd.exithooks = append(cmd.exithooks, hook)
	}

	cmd.Log().Debug().Interface("flags", flags).Msg("flags parsed")

	if err := version.IsValid(nil); err != nil {
		return err
	} else {
		cmd.version = version
	}

	cmd.connectSig()

	return nil
}

func (cmd *BaseCommand) connectSig() {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)

	go func() {
		s := <-sigc

		defer func() {
			os.Exit(1)
		}()

		defer func() {
			cmd.done.Do(cmd.Done)

			_, _ = fmt.Fprintf(os.Stderr, "stopped by force: %v\n", s)
		}()
	}()
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

func (cmd *BaseCommand) LoadEncoders(hinters []hint.Hinter) (*encoder.Encoders, error) {
	if cmd.encs != nil {
		return cmd.encs, nil
	}

	if len(hinters) < 1 {
		hinters = process.DefaultHinters
	}

	ps := pm.NewProcesses().SetContext(context.Background())

	if err := ps.AddProcess(process.ProcessorEncoders, false); err != nil {
		return nil, err
	}

	if err := ps.AddHook(
		pm.HookPrefixPost,
		process.ProcessNameEncoders,
		process.HookNameAddHinters,
		process.HookAddHinters(hinters),
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
