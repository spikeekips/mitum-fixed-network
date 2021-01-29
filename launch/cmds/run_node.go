package cmds

import (
	"context"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/util"
)

var defaultRunHooks = []pm.Hook{
	pm.NewHook(pm.HookPrefixPost, process.ProcessNameLocal, process.HookNameCheckEmptyBlock, process.HookCheckEmptyBlock),
	pm.NewHook(pm.HookPrefixPost, process.ProcessNameConfig, process.HookNameConfigGenesisOperations, pm.EmptyHookFunc).
		SetOverride(true),
}

type RunCommand struct {
	*BaseRunCommand
	ExitAfter time.Duration `name:"exit-after" help:"exit after the given duration"`
}

func NewRunCommand(dryrun bool) RunCommand {
	co := RunCommand{
		BaseRunCommand: NewBaseRunCommand(dryrun, "run"),
	}

	ps := co.Processes()
	for i := range defaultRunHooks {
		hook := defaultRunHooks[i]
		if err := ps.AddHook(hook.Prefix, hook.Process, hook.Name, hook.F, hook.Override); err != nil {
			panic(err)
		}
	}

	_ = co.SetProcesses(ps)

	return co
}

func (cmd *RunCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return xerrors.Errorf("failed to initialize command: %w", err)
	} else {
		defer cmd.Done()
	}

	cmd.Log().Info().Bool("dryrun", cmd.dryrun).Msg("dryrun?")

	if err := cmd.prepare(); err != nil {
		return err
	}

	if cmd.dryrun {
		return nil
	}

	return cmd.run()
}

func (cmd *RunCommand) run() error {
	ps := cmd.Processes()

	var ctx context.Context
	if err := ps.Run(); err != nil {
		return xerrors.Errorf("failed to run: %w", err)
	} else {
		ctx = ps.Context()
	}

	var cs *isaac.ConsensusStates
	if err := process.LoadConsensusStatesContextValue(ctx, &cs); err != nil {
		return err
	}

	select {
	case err := <-cs.ErrChan():
		return err
	case <-func(w time.Duration) <-chan time.Time {
		if w < 1 {
			return make(chan time.Time)
		}

		return time.After(w)
	}(cmd.ExitAfter):

		cmd.Log().Info().Str("exit-after", cmd.ExitAfter.String()).Msg("expired, exit.")

		return nil
	}
}
