package cmds

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/states"
	"github.com/spikeekips/mitum/util"
)

var defaultRunHooks = []pm.Hook{
	pm.NewHook(pm.HookPrefixPre, process.ProcessNameConsensusStates,
		process.HookNameCheckEmptyBlock, process.HookCheckEmptyBlock),
	pm.NewHook(pm.HookPrefixPost, process.ProcessNameConfig,
		process.HookNameConfigGenesisOperations, pm.EmptyHookFunc).
		SetOverride(true),
}

type RunCommand struct {
	*BaseRunCommand
	ExitAfter         time.Duration `name:"exit-after" help:"exit after the given duration"`
	afterStartedHooks *pm.Hooks
}

func NewRunCommand(dryrun bool) RunCommand {
	co := RunCommand{
		BaseRunCommand:    NewBaseRunCommand(dryrun, "run"),
		afterStartedHooks: pm.NewHooks("run-after-started"),
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

	if err := ps.Run(); err != nil {
		return xerrors.Errorf("failed to run: %w", err)
	}

	return cmd.runStates(ps.Context())
}

func (cmd *RunCommand) runStates(ctx context.Context) error {
	var pps *prprocessor.Processors
	if err := process.LoadProposalProcessorContextValue(ctx, &pps); err != nil {
		return err
	}

	var cs states.States
	if err := process.LoadConsensusStatesContextValue(ctx, &cs); err != nil {
		return err
	}

	if err := pps.Start(); err != nil {
		return xerrors.Errorf("failed to start Processors: %w", err)
	}

	errch := make(chan error)
	go func() {
		errch <- cs.Start()
	}()

	if err := cmd.afterStartedHooks.Run(ctx); err != nil {
		return err
	}

	sctx, stopfunc := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP,
	)
	defer stopfunc()

	select {
	case err := <-errch:
		return err
	case <-sctx.Done():
		if err := cs.Stop(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "stop signal received, but failed to stop consensus states, %v\n", err)

			return err
		} else {
			_, _ = fmt.Fprintln(os.Stderr, "stop signal received, consensus states stopped")

			return nil
		}
	case <-func(w time.Duration) <-chan time.Time {
		if w < 1 {
			return make(chan time.Time)
		}

		return time.After(w)
	}(cmd.ExitAfter):
		if err := cs.Stop(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr,
				"expired by exit-after, %v, but failed to stop consensus states: %+v\n", cmd.ExitAfter, err)

			return err
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "expired by exit-after, %v, consensus states stopped\n", cmd.ExitAfter)

			return nil
		}
	}
}

func (cmd *RunCommand) AfterStartedHooks() *pm.Hooks {
	return cmd.afterStartedHooks
}
