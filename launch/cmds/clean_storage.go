package cmds

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util"
)

var ContextValueCleanDatabase util.ContextKey = "clean_database"

type CleanStorageCommand struct {
	*BaseRunCommand
	cleanDatabase func() error
}

func NewCleanStorageCommand(dryrun bool) CleanStorageCommand {
	cmd := CleanStorageCommand{
		BaseRunCommand: NewBaseRunCommand(dryrun, "clean-storage"),
	}

	ps := cmd.Processes()
	if ps == nil {
		panic(errors.Errorf("processes not prepared"))
	}

	disabledProcessors := []string{
		process.ProcessNameProposalProcessor,
		process.ProcessNameConsensusStates,
		process.ProcessNameNetwork,
		process.ProcessNameSuffrage,
		process.ProcessNameTimeSyncer,
	}

	for i := range disabledProcessors {
		if err := ps.RemoveProcess(disabledProcessors[i]); err != nil {
			panic(err)
		}
	}

	hooks := []pm.Hook{
		pm.NewHook(pm.HookPrefixPre, process.ProcessNameGenerateGenesisBlock,
			process.HookNameCheckGenesisBlock, nil),
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameConfig,
			process.HookNameConfigGenesisOperations, nil).SetOverride(true),
	}

	for i := range hooks {
		hook := hooks[i]
		if err := hook.Add(ps); err != nil {
			panic(err)
		}
	}

	_ = cmd.SetProcesses(ps)

	return cmd
}

func (cmd *CleanStorageCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return errors.Wrap(err, "failed to initialize command")
	}
	defer cmd.Done()
	defer func() {
		<-time.After(time.Second * 1)
	}()

	cmd.Log().Info().Bool("dryrun", cmd.dryrun).Msg("dryrun?")

	if err := cmd.prepare(); err != nil {
		return err
	}

	ps := cmd.Processes()
	_ = cmd.SetProcesses(ps)

	if cmd.dryrun {
		return nil
	}

	if err := ps.Run(); err != nil {
		return err
	}

	return cmd.cleanStorage(cmd.Processes().Context())
}

func (cmd *CleanStorageCommand) cleanStorage(ctx context.Context) error {
	var db storage.Database
	if err := process.LoadDatabaseContextValue(ctx, &db); err != nil {
		return err
	}

	var bd blockdata.Blockdata
	if err := process.LoadBlockdataContextValue(ctx, &bd); err != nil {
		return err
	}

	if err := util.LoadFromContextValue(ctx, ContextValueCleanDatabase, &cmd.cleanDatabase); err != nil {
		if !errors.Is(err, util.ContextValueNotFoundError) {
			return err
		}
	}

	if err := blockdata.Clean(db, bd, true); err != nil {
		return err
	}

	if cmd.cleanDatabase != nil {
		if err := cmd.cleanDatabase(); err != nil {
			return err
		}
	}

	cmd.Log().Info().Msg("database and block data was cleaned")

	return nil
}
