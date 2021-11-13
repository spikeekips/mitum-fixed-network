package cmds

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/launch"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/util"
	"gopkg.in/yaml.v3"
)

var (
	defaultConfigLocalNode, _ = base.NewStringAddress("node")
	defaultConfigYAML         = fmt.Sprintf(`
network-id: mitum network; Thu 26 Nov 2020 12:25:18 AM KST
address: %s
privatekey: KzmnCUoBrqYbkoP8AUki1AJsyKqxNsiqdrtTB2onyzQfB6MQ5Sef~btc-priv-v0.0.1
suffrage:
    nodes:
        - %s
`, defaultConfigLocalNode, defaultConfigLocalNode,
	)
)

var DefaultConfigVars = kong.Vars{
	"default_config_default_format": "yaml",
}

type DefaultConfigCommand struct {
	*BaseCommand
	Format string `help:"output format, {yaml}" default:"${default_config_default_format}"`
	out    io.Writer
	ctx    context.Context
}

func NewDefaultConfigCommand() DefaultConfigCommand {
	return DefaultConfigCommand{
		BaseCommand: NewBaseCommand("default_config"),
		out:         os.Stdout,
	}
}

func (cmd *DefaultConfigCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return errors.Wrap(err, "failed to initialize command")
	}

	if err := cmd.prepare(); err != nil {
		return err
	}

	var conf config.LocalNode
	if err := config.LoadConfigContextValue(cmd.ctx, &conf); err != nil {
		return err
	}

	bconf := config.NewBaseLocalNodePackerYAMLFromConfig(conf)
	bconf.Suffrage = map[string]interface{}{
		"type":  "roundrobin",
		"nodes": []base.Address{defaultConfigLocalNode},
	}
	bconf.ProposalProcessor = map[string]interface{}{
		"type": "default",
	}

	b, err := yaml.Marshal(bconf)
	if err != nil {
		return errors.Wrap(err, "failed to format config")
	}
	_, _ = fmt.Fprintln(cmd.out, string(b))

	return nil
}

func (cmd *DefaultConfigCommand) prepare() error {
	switch t := cmd.Format; t {
	case "yaml":
	default:
		return errors.Errorf("unknown output format, %q", t)
	}

	ps := pm.NewProcesses()

	if err := process.Config(ps); err != nil {
		return err
	}

	if err := ps.AddProcess(process.ProcessorEncoders, false); err != nil {
		return err
	}

	if err := ps.AddHook(
		pm.HookPrefixPost,
		process.ProcessNameEncoders,
		process.HookNameAddHinters,
		process.HookAddHinters(launch.EncoderTypes, launch.EncoderHinters),
		true,
	); err != nil {
		return err
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, process.ContextValueConfigSource, []byte(defaultConfigYAML))
	ctx = context.WithValue(ctx, process.ContextValueConfigSourceType, "yaml")
	ctx = context.WithValue(ctx, config.ContextValueLog, cmd.Logging)
	ctx = context.WithValue(ctx, process.ContextValueVersion, cmd.version)

	_ = ps.SetContext(ctx)

	if err := ps.Run(); err != nil {
		return err
	}
	cmd.ctx = ps.Context()

	return nil
}
