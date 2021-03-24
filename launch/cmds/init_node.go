package cmds

import (
	"context"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
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

	if err := ps.AddHook( // NOTE clean database and block data with `--force`
		pm.HookPrefixPre, process.ProcessNameLocalNode,
		"clean-storage", cmd.cleanStorage,
		true,
	); err != nil {
		panic(err)
	}

	for _, i := range []pm.Process{
		process.ProcessorConsensusStates,
		process.ProcessorNetwork,
		process.ProcessorStartNetwork,
	} {
		if err := ps.AddProcess(pm.NewDisabledProcess(i), true); err != nil {
			panic(err)
		}
	}

	if err := ps.AddProcess(process.ProcessorGenerateGenesisBlock, true); err != nil {
		panic(err)
	}
	if err := ps.AddHook(pm.HookPrefixPre, process.ProcessNameGenerateGenesisBlock,
		process.HookNameCheckGenesisBlock, process.HookCheckGenesisBlock, true); err != nil {
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
	var forceCreate bool
	if err := process.LoadGenesisBlockForceCreateContextValue(ctx, &forceCreate); err != nil {
		return ctx, err
	} else if !forceCreate {
		return ctx, nil
	}

	var st storage.Database
	if err := process.LoadDatabaseContextValue(ctx, &st); err != nil {
		return ctx, err
	}

	var blockData blockdata.BlockData
	if err := process.LoadBlockDataContextValue(ctx, &blockData); err != nil {
		return ctx, err
	}

	if err := blockdata.Clean(st, blockData, false); err != nil {
		return ctx, err
	}

	cmd.Log().Info().Msg("database and block data was cleaned by force")

	return ctx, nil
}
