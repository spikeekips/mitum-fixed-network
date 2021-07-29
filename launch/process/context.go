package process

import (
	"context"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/network/discovery"
	"github.com/spikeekips/mitum/network/discovery/memberlist"
	"github.com/spikeekips/mitum/states"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/ulule/limiter/v3"
)

var (
	ContextValueVersion                 util.ContextKey = "version"
	ContextValueConfigSource            util.ContextKey = "config_source"
	ContextValueConfigSourceType        util.ContextKey = "config_source_type"
	ContextValueNetwork                 util.ContextKey = "network"
	ContextValueBlockData               util.ContextKey = "blockdata"
	ContextValueDatabase                util.ContextKey = "database"
	ContextValueLocalNode               util.ContextKey = "local_node"
	ContextValueNodepool                util.ContextKey = "nodepool"
	ContextValueSuffrage                util.ContextKey = "suffrage"
	ContextValueProposalProcessor       util.ContextKey = "proposal_processor"
	ContextValueConsensusStates         util.ContextKey = "consensus_states"
	ContextValueGenesisBlockForceCreate util.ContextKey = "force_create_genesis_block"
	ContextValueGenesisBlock            util.ContextKey = "genesis_block"
	ContextValueOperationProcessors     util.ContextKey = "operation_processors"
	ContextValuePolicy                  util.ContextKey = "policy"
	ContextValueRateLimitStore          util.ContextKey = "ratelimit-store"
	ContextValueRateLimitHandlerMap     util.ContextKey = "ratelimit-handler-map"
	ContextValueDiscovery               util.ContextKey = "discovery"
	ContextValueDiscoveryConnInfos      util.ContextKey = "discovery-conninfos"
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

func LoadBlockDataContextValue(ctx context.Context, l *blockdata.BlockData) error {
	return util.LoadFromContextValue(ctx, ContextValueBlockData, l)
}

func LoadDatabaseContextValue(ctx context.Context, l *storage.Database) error {
	return util.LoadFromContextValue(ctx, ContextValueDatabase, l)
}

func LoadLocalNodeContextValue(ctx context.Context, l **node.Local) error {
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

func LoadRateLimitStoreContextValue(ctx context.Context, l *limiter.Store) error {
	return util.LoadFromContextValue(ctx, ContextValueRateLimitStore, l)
}

func LoadRateLimitHandlerMapContextValue(ctx context.Context, l *map[string][]RateLimitRule) error {
	return util.LoadFromContextValue(ctx, ContextValueRateLimitHandlerMap, l)
}

func LoadDiscoveryContextValue(ctx context.Context, l *discovery.Discovery) error {
	return util.LoadFromContextValue(ctx, ContextValueDiscovery, l)
}

func LoadDiscoveryConnInfosContextValue(ctx context.Context, l *[]memberlist.ConnInfo) error {
	return util.LoadFromContextValue(ctx, ContextValueDiscoveryConnInfos, l)
}
