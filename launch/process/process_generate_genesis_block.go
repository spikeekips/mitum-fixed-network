package process

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
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
		[]string{ProcessNameLocalNode, ProcessNameDatabase, ProcessNameBlockData},
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

	var local *network.LocalNode
	if err := LoadLocalNodeContextValue(ctx, &local); err != nil {
		return ctx, err
	}

	var st storage.Database
	if err := LoadDatabaseContextValue(ctx, &st); err != nil {
		return nil, err
	}

	var blockData blockdata.BlockData
	if err := LoadBlockDataContextValue(ctx, &blockData); err != nil {
		return nil, err
	}

	var policy *isaac.LocalPolicy
	if err := LoadPolicyContextValue(ctx, &policy); err != nil {
		return nil, err
	}

	var l config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &l); err != nil {
		return ctx, err
	}
	ops := l.GenesisOperations()

	log.Debug().Int("operations", len(ops)).Msg("operations loaded")

	if gg, err := isaac.NewGenesisBlockV0Generator(local, st, blockData, policy, ops); err != nil {
		return ctx, xerrors.Errorf("failed to create genesis block generator: %w", err)
	} else if blk, err := gg.Generate(); err != nil {
		return ctx, xerrors.Errorf("failed to generate genesis block: %w", err)
	} else {
		log.Info().
			Dict("block", logging.Dict().Hinted("height", blk.Height()).Hinted("hash", blk.Hash())).
			Msg("genesis block created")

		return context.WithValue(ctx, ContextValueGenesisBlock, blk), nil
	}
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

	var st storage.Database
	if err := LoadDatabaseContextValue(ctx, &st); err != nil {
		return ctx, err
	}

	var blockData blockdata.BlockData
	if err := LoadBlockDataContextValue(ctx, &blockData); err != nil {
		return ctx, err
	}

	var manifest block.Manifest
	if m, found, err := st.LastManifest(); err != nil {
		return ctx, err
	} else if found {
		manifest = m
	}

	if manifest == nil {
		log.Debug().Msg("existing blocks not found")

		return ctx, nil
	}

	log.Debug().Msgf("found existing blocks: block=%d", manifest.Height())

	if !force {
		return ctx, xerrors.Errorf("environment already exists: block=%d", manifest.Height())
	}

	if err := blockdata.Clean(st, blockData, false); err != nil {
		return ctx, err
	}
	log.Debug().Msg("existing environment cleaned")

	return ctx, nil
}
