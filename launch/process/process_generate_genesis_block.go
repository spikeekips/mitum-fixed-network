package process

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/logging"
)

const (
	ProcessNameGenerateGenesisBlock = "generate_genesis_block"
	HookNameCheckGenesisBlock       = "check_genesis_block"
)

var ProcessorGenerateGenesisBlock pm.Process

func init() {
	if i, err := pm.NewProcess(
		ProcessNameGenerateGenesisBlock,
		[]string{ProcessNameLocal, ProcessNameStorage, ProcessNameBlockFS},
		ProcessGenerateGenesisBlock,
	); err != nil {
		panic(err)
	} else {
		ProcessorGenerateGenesisBlock = i
	}
}

func ProcessGenerateGenesisBlock(ctx context.Context) (context.Context, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var local *isaac.Local
	if err := LoadLocalContextValue(ctx, &local); err != nil {
		return ctx, err
	}

	var l config.LocalNode
	var ops []operation.Operation
	if err := config.LoadConfigContextValue(ctx, &l); err != nil {
		return ctx, err
	} else {
		ops = l.GenesisOperations()

		log.Debug().Int("operations", len(ops)).Msg("operations loaded")
	}

	log.Debug().Msg("trying to create genesis block")
	if gg, err := isaac.NewGenesisBlockV0Generator(local, ops); err != nil {
		return ctx, xerrors.Errorf("failed to create genesis block generator: %w", err)
	} else if blk, err := gg.Generate(); err != nil {
		return ctx, xerrors.Errorf("failed to generate genesis block: %w", err)
	} else {
		log.Info().
			Dict("block", logging.Dict().Hinted("height", blk.Height()).Hinted("hash", blk.Hash())).
			Msg("genesis block created")

		ctx = context.WithValue(ctx, ContextValueGenesisBlock, blk)
	}

	log.Info().Msg("genesis block created")
	log.Info().Msg("iniialized")

	return ctx, nil
}

func HookCheckGenesisBlock(ctx context.Context) (context.Context, error) {
	var force bool
	if err := LoadGenesisBlockForceCreateContextValue(ctx, &force); err != nil {
		return ctx, err
	}

	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	log.Debug().Msg("checking existing blocks")

	var st storage.Storage
	if err := LoadStorageContextValue(ctx, &st); err != nil {
		return ctx, err
	}

	var blockFS *storage.BlockFS
	if err := LoadBlockFSContextValue(ctx, &blockFS); err != nil {
		return ctx, err
	}

	var manifest block.Manifest
	if m, found, err := st.LastManifest(); err != nil {
		return ctx, err
	} else if found {
		manifest = m
	}

	if manifest == nil {
		log.Debug().Msg("not found existing blocks")

		return ctx, nil
	}

	log.Debug().Msgf("found existing blocks: block=%d", manifest.Height())

	if !force {
		return ctx, xerrors.Errorf("environment already exists: block=%d", manifest.Height())
	}

	if err := storage.Clean(st, blockFS, false); err != nil {
		return ctx, err
	}

	log.Debug().Msg("existing environment cleaned")

	return ctx, nil
}
