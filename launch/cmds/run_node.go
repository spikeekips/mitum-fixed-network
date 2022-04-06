package cmds

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/deploy"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/network/discovery/memberlist"
	"github.com/spikeekips/mitum/states"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

var defaultRunProcesses = []pm.Process{
	process.ProcessorDiscovery,
	process.ProcessorConsensusStates,
}

var defaultRunHooks = []pm.Hook{
	pm.NewHook(pm.HookPrefixPost, process.ProcessNameDatabase,
		process.HookNameCleanTempMongodbDatabase, process.HookCleanTempMongodbDatabase),
	pm.NewHook(pm.HookPrefixPre, process.ProcessNameConsensusStates,
		process.HookNameCheckEmptyBlock, process.HookCheckEmptyBlock),
	pm.NewHook(pm.HookPrefixPost, process.ProcessNameConfig,
		process.HookNameConfigGenesisOperations, nil).
		SetOverride(true),
	pm.NewHook(pm.HookPrefixPost, process.ProcessNameNetwork,
		deploy.HookNameBlockdataCleaner, deploy.HookBlockdataCleaner),
	pm.NewHook(pm.HookPrefixPost, process.ProcessNameNetwork,
		deploy.HookNameInitializeDeployKeyStorage, deploy.HookInitializeDeployKeyStorage),
	pm.NewHook(pm.HookPrefixPost, process.ProcessNameConsensusStates,
		deploy.HookNameDeployHandlers, deploy.HookDeployHandlers),
}

type RunCommand struct {
	*BaseRunCommand
	Discovery         []*url.URL    `name:"discovery" help:"discovery node"`
	ExitAfter         time.Duration `name:"exit-after" help:"exit after the given duration"`
	NetworkLogFile    []string      `name:"network-log" help:"network log file"`
	afterStartedHooks *pm.Hooks
	cs                states.States
	nt                network.Server
	dis               *memberlist.Discovery
}

func NewRunCommand(dryrun bool) RunCommand {
	co := RunCommand{
		BaseRunCommand:    NewBaseRunCommand(dryrun, "run"),
		afterStartedHooks: pm.NewHooks("run-after-started"),
	}

	ps := co.Processes()
	if err := ps.RemoveProcess(process.ProcessNameConsensusStates); err != nil {
		panic(err)
	}

	for i := range defaultRunProcesses {
		if err := ps.AddProcess(defaultRunProcesses[i], false); err != nil {
			panic(err)
		}
	}

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
		return errors.Wrap(err, "failed to initialize command")
	}
	defer cmd.Done()

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
	l := cmd.Logging
	if len(cmd.NetworkLogFile) > 0 {
		out := cmd.LogOutput
		if len(cmd.NetworkLogFile) > 0 {
			i, err := logging.Outputs(cmd.NetworkLogFile)
			if err != nil {
				return err
			}
			out = i
		}

		l = logging.Setup(out, zerolog.DebugLevel, "json", false)
	}

	networkLogger := logging.NewLogging(func(c zerolog.Context) zerolog.Context {
		return c.Str("module", "network")
	}).SetLogging(l)

	ctx := context.WithValue(cmd.processes.ContextSource(), config.ContextValueNetworkLog, networkLogger)
	ctx = context.WithValue(ctx, config.ContextValueDiscoveryURLs, cmd.Discovery)

	_ = cmd.processes.SetContext(ctx)

	return nil
}

func (cmd *RunCommand) run() error {
	ps := cmd.Processes()

	if err := ps.Run(); err != nil {
		return errors.Wrap(err, "failed to run")
	}

	if err := cmd.runNetwork(ps.Context()); err != nil {
		return errors.Wrap(err, "failed to run network")
	}

	if err := cmd.runDiscovery(ps.Context()); err != nil {
		return errors.Wrap(err, "failed to run discovery")
	}

	if err := cmd.runPPS(ps.Context()); err != nil {
		return errors.Wrap(err, "failed to start Processors")
	}

	return cmd.runStates(ps.Context())
}

func (*RunCommand) runNetwork(ctx context.Context) error {
	var nt network.Server
	if err := process.LoadNetworkContextValue(ctx, &nt); err != nil {
		return err
	}

	return nt.Start()
}

func (*RunCommand) runPPS(ctx context.Context) error {
	var local node.Local
	if err := process.LoadLocalNodeContextValue(ctx, &local); err != nil {
		return err
	}

	var suffrage base.Suffrage
	if err := process.LoadSuffrageContextValue(ctx, &suffrage); err != nil {
		return err
	}

	inSuffrage := suffrage.IsInside(local.Address())

	var pps *prprocessor.Processors
	if !inSuffrage {
		return nil
	}

	if err := process.LoadProposalProcessorContextValue(ctx, &pps); err != nil {
		return err
	}

	return pps.Start()
}

func (cmd *RunCommand) runDiscovery(ctx context.Context) error {
	var local node.Local
	if err := process.LoadLocalNodeContextValue(ctx, &local); err != nil {
		return err
	}

	var suffrage base.Suffrage
	if err := process.LoadSuffrageContextValue(ctx, &suffrage); err != nil {
		return err
	}

	if !suffrage.IsInside(local.Address()) {
		return nil
	}

	var dis *memberlist.Discovery
	if err := util.LoadFromContextValue(ctx, process.ContextValueDiscovery, &dis); err != nil {
		return err
	}

	cmd.dis = dis

	return dis.Start()
}

func (cmd *RunCommand) runStates(ctx context.Context) error {
	var cs states.States
	if err := process.LoadConsensusStatesContextValue(ctx, &cs); err != nil {
		return err
	}

	cmd.cs = cs

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
		if err := cmd.whenExited(); err != nil {
			_, _ = fmt.Fprintf(cmd.LogOutput, "stop signal received, but %+v\n", err)

			return err
		}

		_, _ = fmt.Fprintln(cmd.LogOutput, "stop signal received, consensus states stopped and discovery left")

		return nil
	case <-func(w time.Duration) <-chan time.Time {
		if w < 1 {
			return make(chan time.Time)
		}

		return time.After(w)
	}(cmd.ExitAfter):
		if err := cmd.whenExited(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr,
				"expired by exit-after %v, but %+v\n", cmd.ExitAfter, err)

			return err
		}
		_, _ = fmt.Fprintf(os.Stderr,
			"expired by exit-after, %v, consensus states stopped and discovery left\n", cmd.ExitAfter)

		return nil
	}
}

func (cmd *RunCommand) AfterStartedHooks() *pm.Hooks {
	return cmd.afterStartedHooks
}

func (cmd *RunCommand) whenExited() error {
	if cmd.dis != nil {
		if err := cmd.dis.Leave(time.Second * 10); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "stop signal received, but discovery failed to leave, %v\n", err)

			return errors.Wrap(err, "discovery failed to leave")
		}
	}

	if cmd.nt != nil {
		if err := cmd.nt.Stop(); err != nil {
			return errors.Wrap(err, "failed to stop network.Server")
		}
	}

	if cmd.cs != nil {
		if err := cmd.cs.Stop(); err != nil {
			return errors.Wrap(err, "failed to stop consensus states")
		}
	}

	return nil
}
