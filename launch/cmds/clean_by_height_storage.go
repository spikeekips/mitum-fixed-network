package cmds

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util"
)

const HookNameCleanByHeightStorage = "clean_by_height_storage"

var (
	ContextValueDryRun util.ContextKey = "dry_run"
	ContextValueHeight util.ContextKey = "clean_storage_by_height_height"
)

type CleanByHeightStorageCommand struct {
	*BaseRunCommand
	Height int64 `arg:"" name:"height" help:"height of block" required:"true"`
	DryRun bool  `help:"dry-run" optional:"" default:"false"`
}

func NewCleanByHeightStorageCommand() CleanByHeightStorageCommand {
	cmd := CleanByHeightStorageCommand{
		BaseRunCommand: NewBaseRunCommand(false, "clean-by-height-storage"),
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
		pm.NewHook(pm.HookPrefixPre, process.ProcessNameLocalNode,
			"clean-storage-dry-run", cmd.dryRun),
		pm.NewHook(pm.HookPrefixPre, process.ProcessNameLocalNode,
			HookNameCleanByHeightStorage, cmd.cleanStorage),
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

func (cmd *CleanByHeightStorageCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return errors.Wrap(err, "failed to initialize command")
	}
	defer cmd.Done()

	if err := cmd.prepare(); err != nil {
		return err
	}

	cmd.Log().Info().Bool("dryrun", cmd.DryRun).Int64("height", cmd.Height).Msg("prepared")

	return cmd.Processes().Run()
}

func (cmd *CleanByHeightStorageCommand) prepare() error {
	height := base.Height(cmd.Height)
	if err := height.IsValid(nil); err != nil {
		return err
	}

	if height < base.Height(1) {
		cmd.Log().Warn().Msg("recommend to use clean-storage")
	}

	if err := cmd.BaseRunCommand.prepare(); err != nil {
		return err
	}

	ps := cmd.Processes()
	ps = ps.SetContext(
		context.WithValue(
			context.WithValue(ps.ContextSource(), ContextValueDryRun, cmd.DryRun),
			ContextValueHeight, height),
	)
	_ = cmd.SetProcesses(ps)

	return nil
}

func (cmd *CleanByHeightStorageCommand) dryRun(ctx context.Context) (context.Context, error) {
	var dryrun bool
	switch err := util.LoadFromContextValue(ctx, ContextValueDryRun, &dryrun); {
	case err != nil:
		return ctx, err
	case !dryrun:
		return ctx, nil
	}

	var height base.Height
	if err := util.LoadFromContextValue(ctx, ContextValueHeight, &height); err != nil {
		return ctx, err
	}

	cmd.Log().Debug().Msg("dry-run; will print affected data")

	var db storage.Database
	if err := process.LoadDatabaseContextValue(ctx, &db); err != nil {
		return ctx, err
	}

	var blockData blockdata.BlockData
	if err := process.LoadBlockDataContextValue(ctx, &blockData); err != nil {
		return ctx, err
	}

	var last base.Height
	switch l, ok, err := cmd.check(db, height); {
	case err != nil:
		return ctx, err
	case !ok || height > l:
		_, _ = fmt.Fprintln(os.Stdout, "nothing will be cleaned")

		return ctx, nil
	case height == l:
		_, _ = fmt.Fprintf(os.Stdout, "* %d will be removed.\n", height)

		return ctx, nil
	default:
		last = l
	}

	_, _ = fmt.Fprintf(os.Stdout, "* %d-%d will be removed.\n", height, last)

	return ctx, nil
}

func (cmd *CleanByHeightStorageCommand) cleanStorage(ctx context.Context) (context.Context, error) {
	var dryrun bool
	switch err := util.LoadFromContextValue(ctx, ContextValueDryRun, &dryrun); {
	case err != nil:
		return ctx, err
	case dryrun:
		return ctx, nil
	}

	var db storage.Database
	if err := process.LoadDatabaseContextValue(ctx, &db); err != nil {
		return ctx, err
	}

	var blockData blockdata.BlockData
	if err := process.LoadBlockDataContextValue(ctx, &blockData); err != nil {
		return ctx, err
	}

	var height base.Height
	if err := util.LoadFromContextValue(ctx, ContextValueHeight, &height); err != nil {
		return ctx, err
	}

	switch l, ok, err := cmd.check(db, height); {
	case err != nil:
		return ctx, err
	case !ok || height > l:
		return ctx, nil
	}

	cmd.Log().Debug().Msg("will clean storage by height")

	if err := blockdata.CleanByHeight(db, blockData, height); err != nil {
		return ctx, err
	}

	cmd.Log().Info().Msg("database and block data was cleaned by height")

	return ctx, nil
}

func (cmd *CleanByHeightStorageCommand) check(
	db storage.Database, height base.Height,
) (base.Height, bool, error) {
	switch m, found, err := db.LastManifest(); {
	case err != nil:
		return base.NilHeight, false, err
	case !found:
		return base.NilHeight, false, nil
	case height > m.Height():
		cmd.Log().Debug().Int64("last_block", m.Height().Int64()).Msg("given height is higher than last block")

		return m.Height(), false, nil
	default:
		return m.Height(), true, nil
	}
}
