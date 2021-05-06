package cmds

import (
	"context"

	"github.com/spikeekips/mitum/launch"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
)

type BaseRunCommand struct {
	*BaseCommand
	Design    FileLoad `arg:"" name:"node design file" help:"node design file"`
	dryrun    bool     // NOTE only for testing; it prevent to run node, just prepares
	processes *pm.Processes
}

func NewBaseRunCommand(dryrun bool, name string) *BaseRunCommand {
	return &BaseRunCommand{
		BaseCommand: NewBaseCommand(name),
		dryrun:      dryrun,
		processes:   launch.DefaultProcesses(),
	}
}

func (cmd *BaseRunCommand) Processes() *pm.Processes {
	return cmd.processes
}

func (cmd *BaseRunCommand) SetProcesses(processes *pm.Processes) *BaseRunCommand {
	cmd.processes = processes

	return cmd
}

func (cmd *BaseRunCommand) prepare() error {
	cmd.Log().Info().Msg("prepare to run")

	ctx := context.Background()
	ctx = context.WithValue(ctx, process.ContextValueConfigSource, []byte(cmd.Design))
	ctx = context.WithValue(ctx, process.ContextValueConfigSourceType, "yaml")
	ctx = context.WithValue(ctx, config.ContextValueLog, cmd.Log())
	ctx = context.WithValue(ctx, process.ContextValueVersion, cmd.version)

	ps := cmd.Processes()
	_ = ps.SetContext(ctx)
	_ = ps.SetLogger(cmd.Log())

	_ = cmd.SetProcesses(ps)

	cmd.Log().Info().Msg("prepared")

	return nil
}
