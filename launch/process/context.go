package process

import (
	"context"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/states"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
)

var (
	ContextValueVersion                 util.ContextKey = "version"
	ContextValueConfigSource            util.ContextKey = "config_source"
	ContextValueConfigSourceType        util.ContextKey = "config_source_type"
	ContextValueLog                     util.ContextKey = "log"
	ContextValueNetwork                 util.ContextKey = "network"
	ContextValueBlockFS                 util.ContextKey = "blockfs"
	ContextValueStorage                 util.ContextKey = "storage"
	ContextValueLocalNode               util.ContextKey = "local_node"
	ContextValueNodepool                util.ContextKey = "nodepool"
	ContextValueSuffrage                util.ContextKey = "suffrage"
	ContextValueProposalProcessor       util.ContextKey = "proposal_processor"
	ContextValueConsensusStates         util.ContextKey = "consensus_states"
	ContextValueGenesisBlockForceCreate util.ContextKey = "force_create_genesis_block"
	ContextValueGenesisBlock            util.ContextKey = "genesis_block"
	ContextValueOperationProcessors     util.ContextKey = "operation_processors"
	ContextValuePolicy                  util.ContextKey = "policy"
)

func LoadConfigSourceContextValue(ctx context.Context, l *[]byte) error {
	return util.LoadFromContextValue(ctx, ContextValueConfigSource, l)
}

func LoadConfigSourceTypeContextValue(ctx context.Context, l *string) error {
	return util.LoadFromContextValue(ctx, ContextValueConfigSourceType, l)
}

func LoadVersionContextValue(ctx context.Context, l *util.Version) error {
	return util.LoadFromContextValue(ctx, ContextValueVersion, l)
}

func LoadNetworkContextValue(ctx context.Context, l *network.Server) error {
	return util.LoadFromContextValue(ctx, ContextValueNetwork, l)
}

func LoadBlockFSContextValue(ctx context.Context, l **storage.BlockFS) error {
	return util.LoadFromContextValue(ctx, ContextValueBlockFS, l)
}

func LoadStorageContextValue(ctx context.Context, l *storage.Storage) error {
	return util.LoadFromContextValue(ctx, ContextValueStorage, l)
}

func LoadLocalNodeContextValue(ctx context.Context, l **network.LocalNode) error {
	return util.LoadFromContextValue(ctx, ContextValueLocalNode, l)
}

func LoadNodepoolContextValue(ctx context.Context, l **network.Nodepool) error {
	return util.LoadFromContextValue(ctx, ContextValueNodepool, l)
}

func LoadSuffrageContextValue(ctx context.Context, l *base.Suffrage) error {
	return util.LoadFromContextValue(ctx, ContextValueSuffrage, l)
}

func LoadProposalProcessorContextValue(ctx context.Context, l **prprocessor.Processors) error {
	return util.LoadFromContextValue(ctx, ContextValueProposalProcessor, l)
}

func LoadConsensusStatesContextValue(ctx context.Context, l *states.States) error {
	return util.LoadFromContextValue(ctx, ContextValueConsensusStates, l)
}

func LoadGenesisBlockForceCreateContextValue(ctx context.Context, l *bool) error {
	return util.LoadFromContextValue(ctx, ContextValueGenesisBlockForceCreate, l)
}

func LoadGenesisBlockContextValue(ctx context.Context, l *block.Block) error {
	return util.LoadFromContextValue(ctx, ContextValueGenesisBlock, l)
}

func LoadOperationProcessorsContextValue(ctx context.Context, l **hint.Hintmap) error {
	return util.LoadFromContextValue(ctx, ContextValueOperationProcessors, l)
}

func LoadPolicyContextValue(ctx context.Context, l **isaac.LocalPolicy) error {
	return util.LoadFromContextValue(ctx, ContextValuePolicy, l)
}
