package cmds

import (
	"context"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/storage/blockdata/localfs"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"golang.org/x/xerrors"
)

type RestoreCommand struct {
	*BaseRunCommand
	CleanDatabase bool `name:"clean-database" help:"clean database"`
	bd            *localfs.BlockData
	st            storage.Database
	policy        *isaac.LocalPolicy
	lastBlock     block.Block
	lastManifest  block.Manifest
}

func NewRestoreCommand(types []hint.Type, hinters []hint.Hinter) RestoreCommand {
	cmd := RestoreCommand{
		BaseRunCommand: NewBaseRunCommand(false, "restore"),
	}

	ps := cmd.Processes()
	if ps == nil {
		panic(xerrors.Errorf("processes not prepared"))
	}

	for _, i := range []pm.Process{
		process.ProcessorConsensusStates,
		process.ProcessorNetwork,
		process.ProcessorProposalProcessor,
		process.ProcessorSuffrage,
	} {
		if err := ps.AddProcess(pm.NewDisabledProcess(i), true); err != nil {
			panic(err)
		}
	}

	restoreHooks := []pm.Hook{
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameEncoders,
			process.HookNameAddHinters, process.HookAddHinters(types, hinters)).SetOverride(true),
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameConfig,
			process.HookNameConfigGenesisOperations, nil).SetOverride(true),
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameConfig,
			process.HookNameConfigGenesisOperations, nil).SetOverride(true),
		pm.NewHook(pm.HookPrefixPre, process.ProcessNameBlockData,
			process.HookNameCheckBlockDataPath, nil).SetOverride(true),
	}

	for i := range restoreHooks {
		hook := restoreHooks[i]
		if err := ps.AddHook(hook.Prefix, hook.Process, hook.Name, hook.F, hook.Override); err != nil {
			panic(err)
		}
	}

	_ = cmd.SetProcesses(ps)

	return cmd
}

func (cmd *RestoreCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return xerrors.Errorf("failed to initialize command: %w", err)
	}
	defer cmd.Done()
	defer func() {
		<-time.After(time.Second * 1)
	}()

	if err := cmd.prepare(); err != nil {
		return err
	}

	ps := cmd.Processes()
	if err := ps.Run(); err != nil {
		return err
	}

	return cmd.restore()
}

func (cmd *RestoreCommand) restore() error {
	cmd.Log().Debug().Msg("trying to restore")

	var height base.Height = base.PreGenesisHeight
	if cmd.lastManifest != nil {
		height = cmd.lastManifest.Height() + 1
	}
	for {
		if found, err := cmd.bd.Exists(height); err != nil {
			return xerrors.Errorf("failed to check blockdata of height, %d: %w", height, err)
		} else if !found {
			break
		}

		if err := cmd.restoreBlock(height); err != nil {
			return err
		}

		cmd.Log().Info().Int64("height", height.Int64()).Msg("block restored")

		if height == cmd.lastBlock.Height() {
			break
		}

		height++
	}

	cmd.Log().Info().Msg("restored")

	return nil
}

func (cmd *RestoreCommand) restoreBlock(height base.Height) error {
	sst, err := cmd.st.NewSyncerSession()
	if err != nil {
		return err
	}
	defer func() {
		_ = sst.Close()
	}()

	l := cmd.Log().With().Int64("height", height.Int64()).Logger()

	var blk block.Block
	var bdm block.BaseBlockDataMap
	if i, j, err := localfs.LoadBlock(cmd.bd, height); err != nil {
		l.Error().Err(err).Msg("failed to load block")

		return err
	} else if err := j.IsValid(cmd.policy.NetworkID()); err != nil {
		l.Error().Err(err).Msg("invalid block")

		return err
	} else {
		blk = j
		bdm = i
	}

	if err := sst.SetBlocks([]block.Block{blk}, []block.BlockDataMap{bdm}); err != nil {
		return err
	}

	if err := sst.Commit(); err != nil {
		return err
	} else if st, ok := cmd.st.(storage.LastBlockSaver); ok {
		if err := st.SaveLastBlock(height); err != nil {
			return err
		}
	}

	return nil
}

func (cmd *RestoreCommand) prepare() error {
	if err := cmd.BaseRunCommand.prepare(); err != nil {
		return err
	}

	ps := cmd.Processes()

	hooks := []pm.Hook{
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameLocalNode,
			"load-vars", cmd.hookLoadVars),
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameLocalNode,
			"check-empty-blockdata", cmd.hookCheckEmptyBlockData),
	}

	if cmd.CleanDatabase {
		hooks = append(hooks, pm.NewHook(pm.HookPrefixPost, process.ProcessNameLocalNode,
			"clean-database", cmd.hookCleanDatabase),
		)
	} else {
		hooks = append(hooks, pm.NewHook(pm.HookPrefixPost, process.ProcessNameLocalNode,
			"check-database", cmd.hookCheckExistingDatabase),
		)
	}

	for i := range hooks {
		hook := hooks[i]
		if err := ps.AddHook(hook.Prefix, hook.Process, hook.Name, hook.F, hook.Override); err != nil {
			panic(err)
		}
	}

	return nil
}

func (cmd *RestoreCommand) hookLoadVars(ctx context.Context) (context.Context, error) {
	var bd blockdata.BlockData
	if err := process.LoadBlockDataContextValue(ctx, &bd); err != nil {
		return ctx, err
	} else if i, ok := bd.(*localfs.BlockData); !ok {
		return ctx, util.WrongTypeError.Errorf("BlockData is not type of *localfs.BlockData, %T", bd)
	} else {
		cmd.bd = i
	}

	var st storage.Database
	if err := process.LoadDatabaseContextValue(ctx, &st); err != nil {
		return ctx, err
	}
	cmd.st = st

	var policy *isaac.LocalPolicy
	if err := process.LoadPolicyContextValue(ctx, &policy); err != nil {
		return ctx, err
	}
	cmd.policy = policy

	return ctx, nil
}

func (cmd *RestoreCommand) hookCheckEmptyBlockData(ctx context.Context) (context.Context, error) {
	var height base.Height = base.PreGenesisHeight
	for {
		if found, err := cmd.bd.Exists(height); err != nil {
			return ctx, xerrors.Errorf("failed to check blockdata of height, %d: %w", height, err)
		} else if !found {
			height--
			break
		}

		height++
	}

	if height < base.PreGenesisHeight+1 {
		return ctx, xerrors.Errorf("blockdata is empty")
	}

	cmd.Log().Debug().Int64("last_height", height.Int64()).Msg("blockdata checked")

	_, i, err := localfs.LoadBlock(cmd.bd, height)
	if err != nil {
		return ctx, err
	}
	cmd.lastBlock = i

	return ctx, nil
}

func (cmd *RestoreCommand) hookCheckExistingDatabase(ctx context.Context) (context.Context, error) {
	switch m, found, err := cmd.st.LastManifest(); {
	case err != nil:
		return ctx, err
	case !found:
		cmd.Log().Debug().Msg("last manifest not found")
		return ctx, nil
	case m != nil:
		cmd.Log().Debug().Object("block", m).Msg("last manfest found in database; restore from it")

		cmd.lastManifest = m
	}

	switch {
	case cmd.lastManifest.Height() > cmd.lastBlock.Height():
		return ctx,
			xerrors.Errorf("block in database is higher than blockdata; clean database first with --clean-database")
	case cmd.lastManifest.Height() == cmd.lastBlock.Height():
		if !cmd.lastManifest.Hash().Equal(cmd.lastBlock.Hash()) {
			return ctx, xerrors.Errorf("block in database has same height with blockdata, " +
				"but different hash; clean database first with --clean-database")
		}

		return ctx, util.IgnoreError.Errorf("block in database is already same with blockdata")
	default:
		if _, j, err := localfs.LoadBlock(cmd.bd, cmd.lastManifest.Height()); err != nil {
			return ctx, xerrors.Errorf("failed to load block of last manifest")
		} else if !j.Hash().Equal(cmd.lastManifest.Hash()) {
			return ctx, xerrors.Errorf("hash of last manifest does not match with one of blockdata" +
				"; clean database first with --clean-database")
		}
	}

	return ctx, nil
}

func (cmd *RestoreCommand) hookCleanDatabase(ctx context.Context) (context.Context, error) {
	if err := cmd.st.Clean(); err != nil {
		return ctx, err
	}

	cmd.Log().Debug().Msg("database cleaned")

	return ctx, cmd.st.Clean()
}
