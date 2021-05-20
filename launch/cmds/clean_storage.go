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

type CleanStorageCommand struct {
	*BaseRunCommand
}

func NewCleanStorageCommand(dryrun bool) CleanStorageCommand {
	cmd := CleanStorageCommand{
		BaseRunCommand: NewBaseRunCommand(dryrun, "clean-storage"),
	}

	ps := cmd.Processes()
	if ps == nil {
		panic(xerrors.Errorf("processes not prepared"))
	}

	if err := ps.AddHook( // NOTE clean storage and block data with `--force`
		pm.HookPrefixPre, process.ProcessNameLocalNode,
		"clean-storage", cmd.cleanStorage,
		true,
	); err != nil {
		panic(err)
	}

	disabledProcessors := []string{
		process.ProcessNameStartNetwork,
		process.ProcessNameProposalProcessor,
		process.ProcessNameConsensusStates,
		process.ProcessNameNetwork,
		process.ProcessNameSuffrage,
	}

	for i := range disabledProcessors {
		if err := ps.RemoveProcess(disabledProcessors[i]); err != nil {
			panic(err)
		}
	}

	if err := ps.AddHook(
		pm.HookPrefixPre,
		process.ProcessNameGenerateGenesisBlock,
		process.HookNameCheckGenesisBlock,
		nil,
		true,
	); err != nil {
		panic(err)
	}

	_ = cmd.SetProcesses(ps)

	return cmd
}

func (cmd *CleanStorageCommand) Run(version util.Version) error {
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
	_ = cmd.SetProcesses(ps)

	if cmd.dryrun {
		return nil
	}

	return ps.Run()
}

func (cmd *CleanStorageCommand) cleanStorage(ctx context.Context) (context.Context, error) {
	var st storage.Database
	if err := process.LoadDatabaseContextValue(ctx, &st); err != nil {
		return ctx, err
	}

	var blockData blockdata.BlockData
	if err := process.LoadBlockDataContextValue(ctx, &blockData); err != nil {
		return ctx, err
	}

	if err := blockdata.Clean(st, blockData, true); err != nil {
		return ctx, err
	}

	cmd.Log().Info().Msg("database and block data was cleaned")

	return ctx, nil
}
