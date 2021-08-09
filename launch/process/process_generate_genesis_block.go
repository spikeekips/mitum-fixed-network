package process

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
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
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var local *node.Local
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

	log.Log().Debug().Int("operations", len(ops)).Msg("operations loaded")

	if gg, err := isaac.NewGenesisBlockV0Generator(local, st, blockData, policy, ops); err != nil {
		return ctx, errors.Wrap(err, "failed to create genesis block generator")
	} else if blk, err := gg.Generate(); err != nil {
		return ctx, errors.Wrap(err, "failed to generate genesis block")
	} else {
		log.Log().Info().Object("block", blk).Msg("genesis block created")

		return context.WithValue(ctx, ContextValueGenesisBlock, blk), nil
	}
}

func HookCheckGenesisBlock(ctx context.Context) (context.Context, error) {
	var force bool
	if err := LoadGenesisBlockForceCreateContextValue(ctx, &force); err != nil {
		return ctx, err
	}

	var log *logging.Logging
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
		log.Log().Debug().Msg("existing blocks not found")

		return ctx, nil
	}

	log.Log().Debug().Msgf("found existing blocks: block=%d", manifest.Height())

	if !force {
		return ctx, errors.Errorf("environment already exists: block=%d", manifest.Height())
	}

	if err := blockdata.Clean(st, blockData, false); err != nil {
		return ctx, err
	}
	log.Log().Debug().Msg("existing environment cleaned")

	return ctx, nil
}
