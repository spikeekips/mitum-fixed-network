package cmds

import (
	"context"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
)

type InitCommand struct {
	*BaseRunCommand
	Force bool `help:"clean the existing environment"`
}

func NewInitCommand(dryrun bool) InitCommand {
	cmd := InitCommand{
		BaseRunCommand: NewBaseRunCommand(dryrun, "init"),
	}

	ps := cmd.Processes()
	if ps == nil {
		panic(xerrors.Errorf("processes not prepared"))
	}

	if err := ps.AddHook( // NOTE clean storage and blockfs with `--force`
		pm.HookPrefixPre, process.ProcessNameLocal,
		"clean-storage", cmd.cleanStorage,
		true,
	); err != nil {
		panic(err)
	}

	if err := ps.AddProcess(pm.NewDisabledProcess(process.ProcessorStartNetwork), true); err != nil {
		panic(err)
	}
	if err := ps.AddProcess(pm.NewDisabledProcess(process.ProcessorStartConsensusStates), true); err != nil {
		panic(err)
	}

	if err := ps.AddProcess(process.ProcessorGenerateGenesisBlock, true); err != nil {
		panic(err)
	}
	if err := ps.AddHook(
		pm.HookPrefixPre,
		process.ProcessNameGenerateGenesisBlock,
		process.HookNameCheckGenesisBlock,
		process.HookCheckGenesisBlock,
		true,
	); err != nil {
		panic(err)
	}

	_ = cmd.SetProcesses(ps)

	return cmd
}

func (cmd *InitCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return xerrors.Errorf("failed to initialize command: %w", err)
	} else {
		defer cmd.Done()
		defer func() {
			<-time.After(time.Second * 1)
		}()
	}

	cmd.Log().Info().Bool("dryrun", cmd.dryrun).Msg("dryrun?")

	if err := cmd.prepare(); err != nil {
		return err
	}

	ps := cmd.Processes()
	_ = ps.SetContext(context.WithValue(ps.ContextSource(), process.ContextValueGenesisBlockForceCreate, cmd.Force))
	_ = cmd.SetProcesses(ps)

	if cmd.dryrun {
		return nil
	}

	return ps.Run()
}

func (cmd *InitCommand) cleanStorage(ctx context.Context) (context.Context, error) {
	var force bool
	if err := process.LoadGenesisBlockForceCreateContextValue(ctx, &force); err != nil {
		return ctx, err
	} else if !force {
		return ctx, nil
	}

	var st storage.Storage
	if err := process.LoadStorageContextValue(ctx, &st); err != nil {
		return ctx, err
	}

	var blockFS *storage.BlockFS
	if err := process.LoadBlockFSContextValue(ctx, &blockFS); err != nil {
		return ctx, err
	}

	if err := storage.Clean(st, blockFS, false); err != nil {
		return ctx, err
	}

	cmd.Log().Info().Msg("storage and blockfs was cleaned by force")

	return ctx, nil
}
