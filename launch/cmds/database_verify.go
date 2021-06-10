package cmds

import (
	"context"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/launch"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

var (
	networkIDContextKey    util.ContextKey = "network-id"
	lastManifestContextKey util.ContextKey = "last-manifest"
)

var databaseVerifyProcesses = []pm.Process{
	process.ProcessorTimeSyncer,
	process.ProcessorEncoders,
	process.ProcessorDatabase,
	process.ProcessorBlockData,
}

var databaseVerifyHooks = []pm.Hook{
	pm.NewHook(pm.HookPrefixPost, process.ProcessNameEncoders,
		process.HookNameAddHinters, process.HookAddHinters(launch.EncoderTypes, launch.EncoderHinters)),
	pm.NewHook(pm.HookPrefixPre, process.ProcessNameBlockData,
		process.HookNameCheckBlockDataPath, process.HookCheckBlockDataPath),
	pm.NewHook(pm.HookPrefixPost, process.ProcessNameBlockData,
		"check_storage", hookCheckStorage),
}

func init() {
	if i, err := pm.NewProcess(
		process.ProcessNameConfig,
		nil,
		pm.EmptyProcessFunc,
	); err != nil {
		panic(err)
	} else {
		databaseVerifyProcesses = append(databaseVerifyProcesses, i)
	}
}

type DatabaseVerifyCommand struct {
	*BaseVerifyCommand
	URI          string `arg:"" name:"database uri"`
	Path         string `arg:"" name:"blockdata path"`
	processes    *pm.Processes
	database     storage.Database
	blockData    blockdata.BlockData
	lastManifest block.Manifest
}

func NewDatabaseVerifyCommand(types []hint.Type, hinters []hint.Hinter) DatabaseVerifyCommand {
	return DatabaseVerifyCommand{
		BaseVerifyCommand: NewBaseVerifyCommand("database-verify", types, hinters),
	}
}

func (cmd *DatabaseVerifyCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return xerrors.Errorf("failed to initialize command: %w", err)
	}

	cmd.Log().Debug().Str("uri", cmd.URI).Str("path", cmd.Path).Msg("trying to verify database")

	return cmd.verify()
}

func (cmd *DatabaseVerifyCommand) Initialize(flags interface{}, version util.Version) error {
	if err := cmd.BaseVerifyCommand.Initialize(flags, version); err != nil {
		return err
	}

	if i, err := cmd.initializeProcesses(); err != nil {
		return err
	} else if err := i.Run(); err != nil {
		return err
	} else {
		cmd.processes = i
	}

	return cmd.prepare()
}

func (cmd *DatabaseVerifyCommand) verify() error {
	cmd.Log().Debug().Msg("verifying database")
	if err := cmd.checkAllManifests(cmd.loadManifest); err != nil {
		return err
	}

	cmd.Log().Info().Msg("database verified")

	return nil
}

func (cmd *DatabaseVerifyCommand) loadManifest(height base.Height) (block.Manifest, error) {
	switch i, found, err := cmd.database.ManifestByHeight(height); {
	case err != nil:
		return nil, err
	case !found:
		return nil, util.NotFoundError.Errorf("manifest, %d not found", height)
	default:
		return i, nil
	}
}

func (cmd *DatabaseVerifyCommand) initializeProcesses() (*pm.Processes, error) {
	conf := config.NewBaseLocalNode(jsonenc.NewEncoder(), nil)
	if err := conf.Storage().Database().SetURI(cmd.URI); err != nil {
		return nil, err
	} else if err := conf.Storage().BlockData().SetPath(cmd.Path); err != nil {
		return nil, err
	}

	ctx := context.WithValue(context.Background(), config.ContextValueConfig, conf)
	ctx = context.WithValue(ctx, config.ContextValueLog, cmd.Log())
	ctx = context.WithValue(ctx, networkIDContextKey, cmd.networkID)

	ps := pm.NewProcesses()
	_ = ps.SetContext(ctx)
	_ = ps.SetLogger(cmd.Log())

	for i := range databaseVerifyProcesses {
		if err := ps.AddProcess(databaseVerifyProcesses[i], false); err != nil {
			return nil, err
		}
	}

	for i := range databaseVerifyHooks {
		hook := databaseVerifyHooks[i]
		if err := ps.AddHook(hook.Prefix, hook.Process, hook.Name, hook.F, true); err != nil {
			return nil, err
		}
	}

	return ps, nil
}

func (cmd *DatabaseVerifyCommand) prepare() error {
	ctx := cmd.processes.Context()

	var database storage.Database
	if err := process.LoadDatabaseContextValue(ctx, &database); err != nil {
		return err
	}
	cmd.database = database

	var blockData blockdata.BlockData
	if err := process.LoadBlockDataContextValue(ctx, &blockData); err != nil {
		return err
	}
	cmd.blockData = blockData

	var lastManifest block.Manifest
	if err := util.LoadFromContextValue(ctx, lastManifestContextKey, &lastManifest); err != nil {
		return err
	}
	cmd.lastHeight = lastManifest.Height()
	cmd.lastManifest = lastManifest

	return nil
}

func hookCheckStorage(ctx context.Context) (context.Context, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var db storage.Database
	if err := process.LoadDatabaseContextValue(ctx, &db); err != nil {
		return ctx, err
	}

	var blockData blockdata.BlockData
	if err := process.LoadBlockDataContextValue(ctx, &blockData); err != nil {
		return ctx, err
	}

	var networkID base.NetworkID
	if err := util.LoadFromContextValue(ctx, networkIDContextKey, &networkID); err != nil {
		return ctx, err
	}

	i, err := blockdata.CheckBlock(db, blockData, networkID)
	if err != nil {
		return ctx, err
	}
	log.Debug().Dict("block", logging.Dict().
		Hinted("hash", i.Hash()).
		Hinted("height", i.Height()).
		Hinted("round", i.Round())).
		Msg("block found")

	ctx = context.WithValue(ctx, lastManifestContextKey, i)

	return ctx, nil
}
