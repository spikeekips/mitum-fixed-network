package process

import (
	"context"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
)

var (
	ContextValueVersion                 config.ContextKey = "version"
	ContextValueConfigSource            config.ContextKey = "config_source"
	ContextValueConfigSourceType        config.ContextKey = "config_source_type"
	ContextValueLog                     config.ContextKey = "log"
	ContextValueNetwork                 config.ContextKey = "network"
	ContextValueBlockFS                 config.ContextKey = "blockfs"
	ContextValueStorage                 config.ContextKey = "storage"
	ContextValueLocal                   config.ContextKey = "local"
	ContextValueSuffrage                config.ContextKey = "suffrage"
	ContextValueProposalProcessor       config.ContextKey = "proposal_processor"
	ContextValueConsensusStates         config.ContextKey = "consensus_states"
	ContextValueGenesisBlockForceCreate config.ContextKey = "force_create_genesis_block"
	ContextValueGenesisBlock            config.ContextKey = "genesis_block"
	ContextValueOperationProcessors     config.ContextKey = "operation_processors"
)

func LoadConfigSourceContextValue(ctx context.Context, l *[]byte) error {
	return config.LoadFromContextValue(ctx, ContextValueConfigSource, l)
}

func LoadConfigSourceTypeContextValue(ctx context.Context, l *string) error {
	return config.LoadFromContextValue(ctx, ContextValueConfigSourceType, l)
}

func LoadVersionContextValue(ctx context.Context, l *util.Version) error {
	return config.LoadFromContextValue(ctx, ContextValueVersion, l)
}

func LoadNetworkContextValue(ctx context.Context, l *network.Server) error {
	return config.LoadFromContextValue(ctx, ContextValueNetwork, l)
}

func LoadBlockFSContextValue(ctx context.Context, l **storage.BlockFS) error {
	return config.LoadFromContextValue(ctx, ContextValueBlockFS, l)
}

func LoadStorageContextValue(ctx context.Context, l *storage.Storage) error {
	return config.LoadFromContextValue(ctx, ContextValueStorage, l)
}

func LoadLocalContextValue(ctx context.Context, l **isaac.Local) error {
	return config.LoadFromContextValue(ctx, ContextValueLocal, l)
}

func LoadSuffrageContextValue(ctx context.Context, l *base.Suffrage) error {
	return config.LoadFromContextValue(ctx, ContextValueSuffrage, l)
}

func LoadProposalProcessorContextValue(ctx context.Context, l **prprocessor.Processors) error {
	return config.LoadFromContextValue(ctx, ContextValueProposalProcessor, l)
}

func LoadConsensusStatesContextValue(ctx context.Context, l **isaac.ConsensusStates) error {
	return config.LoadFromContextValue(ctx, ContextValueConsensusStates, l)
}

func LoadGenesisBlockForceCreateContextValue(ctx context.Context, l *bool) error {
	return config.LoadFromContextValue(ctx, ContextValueGenesisBlockForceCreate, l)
}

func LoadGenesisBlockContextValue(ctx context.Context, l *block.Block) error {
	return config.LoadFromContextValue(ctx, ContextValueGenesisBlock, l)
}

func LoadOperationProcessorsContextValue(ctx context.Context, l **hint.Hintmap) error {
	return config.LoadFromContextValue(ctx, ContextValueOperationProcessors, l)
}
