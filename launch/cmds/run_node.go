package cmds

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/xerrors"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/states"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

var defaultRunHooks = []pm.Hook{
	pm.NewHook(pm.HookPrefixPre, process.ProcessNameConsensusStates,
		process.HookNameCheckEmptyBlock, process.HookCheckEmptyBlock),
	pm.NewHook(pm.HookPrefixPost, process.ProcessNameConfig,
		process.HookNameConfigGenesisOperations, nil).
		SetOverride(true),
}

type RunCommand struct {
	*BaseRunCommand
	ExitAfter         time.Duration `name:"exit-after" help:"exit after the given duration"`
	NetworkLogFile    []string      `name:"network-log" help:"network log file"`
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

func (cmd *RunCommand) prepare() error {
	if err := cmd.BaseRunCommand.prepare(); err != nil {
		return err
	}

	// NOTE setup network log
	var networkLogger logging.Logger
	if len(cmd.NetworkLogFile) < 1 {
		networkLogger = cmd.Log()
	} else {
		outs := make([]io.Writer, len(cmd.NetworkLogFile))
		for i, f := range cmd.NetworkLogFile {
			if out, err := LogOutput(f); err != nil {
				return err
			} else {
				outs[i] = out
			}
		}

		networkLogger = SetupLogging(
			zerolog.MultiLevelWriter(outs...),
			zerolog.DebugLevel, "json", true, false,
		)
	}

	ctx := context.WithValue(cmd.processes.ContextSource(), config.ContextValueNetworkLog, networkLogger)
	_ = cmd.processes.SetContext(ctx)

	return nil
}

func (cmd *RunCommand) run() error {
	ps := cmd.Processes()

	if err := ps.Run(); err != nil {
		return xerrors.Errorf("failed to run: %w", err)
	}

	return cmd.runStates(ps.Context())
}

func (cmd *RunCommand) prepareStates(ctx context.Context) (states.States, error) {
	var cs states.States
	if err := process.LoadConsensusStatesContextValue(ctx, &cs); err != nil {
		return nil, err
	}

	var nodepool *network.Nodepool
	if err := process.LoadNodepoolContextValue(ctx, &nodepool); err != nil {
		return nil, err
	}

	var suffrage base.Suffrage
	if err := process.LoadSuffrageContextValue(ctx, &suffrage); err != nil {
		return nil, err
	}

	if suffrage.IsInside(nodepool.Local().Address()) {
		var pps *prprocessor.Processors
		if err := process.LoadProposalProcessorContextValue(ctx, &pps); err != nil {
			return nil, err
		}

		if err := pps.Start(); err != nil {
			return nil, xerrors.Errorf("failed to start Processors: %w", err)
		}
	}

	return cs, nil
}

func (cmd *RunCommand) runStates(ctx context.Context) error {
	var cs states.States
	if i, err := cmd.prepareStates(ctx); err != nil {
		return err
	} else {
		cs = i
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
