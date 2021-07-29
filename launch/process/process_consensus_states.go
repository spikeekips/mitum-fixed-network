package process

import (
	"context"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/states"
	basicstate "github.com/spikeekips/mitum/states/basic"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util/logging"
)

const ProcessNameConsensusStates = "consensus_states"

var ProcessorConsensusStates pm.Process

func init() {
	if i, err := pm.NewProcess(
		ProcessNameConsensusStates,
		[]string{
			ProcessNameLocalNode,
			ProcessNameDatabase,
			ProcessNameBlockData,
			ProcessNameSuffrage,
			ProcessNameProposalProcessor,
		},
		ProcessConsensusStates,
	); err != nil {
		panic(err)
	} else {
		ProcessorConsensusStates = i
	}
}

func ProcessConsensusStates(ctx context.Context) (context.Context, error) {
	var policy *isaac.LocalPolicy
	if err := LoadPolicyContextValue(ctx, &policy); err != nil {
		return ctx, err
	}

	var nodepool *network.Nodepool
	if err := LoadNodepoolContextValue(ctx, &nodepool); err != nil {
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

	var suffrage base.Suffrage
	if err := LoadSuffrageContextValue(ctx, &suffrage); err != nil {
		return ctx, err
	}

	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var cs states.States
	if suffrage.IsInside(nodepool.LocalNode().Address()) {
		i, err := processConsensusStatesSuffrageNode(ctx, st, blockData, policy, nodepool, suffrage)
		if err != nil {
			return ctx, err
		}
		cs = i
	} else {
		i, err := processConsensusStatesNoneSuffrageNode(ctx, st, blockData, policy, nodepool, suffrage)
		if err != nil {
			return ctx, err
		}
		cs = i
	}

	if i, ok := cs.(logging.SetLogger); ok {
		_ = i.SetLogger(log)
	}

	return context.WithValue(ctx, ContextValueConsensusStates, cs), nil
}

func processConsensusStatesSuffrageNode(
	ctx context.Context,
	st storage.Database,
	blockData blockdata.BlockData,
	policy *isaac.LocalPolicy,
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
) (states.States, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return nil, err
	}

	var pps *prprocessor.Processors
	if err := LoadProposalProcessorContextValue(ctx, &pps); err != nil {
		return nil, err
	}

	log.Debug().Msg("local is in suffrage")

	proposalMaker := isaac.NewProposalMaker(nodepool.LocalNode(), st, policy)

	ballotbox := isaac.NewBallotbox(
		suffrage.Nodes,
		func() base.Threshold {
			t, err := base.NewThreshold(
				uint(len(suffrage.Nodes())),
				policy.ThresholdRatio(),
			)
			if err != nil {
				panic(err)
			}
			return t
		},
	)
	_ = ballotbox.SetLogger(log)

	stopped := basicstate.NewStoppedState()
	booting := basicstate.NewBootingState(nodepool.LocalNode(), st, blockData, policy, suffrage)
	joining := basicstate.NewJoiningState(nodepool.LocalNode(), st, policy, suffrage, ballotbox)
	consensus := basicstate.NewConsensusState(st, policy, nodepool, suffrage, proposalMaker, pps)
	syncing := basicstate.NewSyncingState(st, blockData, policy, nodepool)

	return basicstate.NewStates(
		st,
		policy,
		nodepool,
		suffrage,
		ballotbox,
		stopped,
		booting,
		joining,
		consensus,
		syncing,
	)
}

func processConsensusStatesNoneSuffrageNode(
	ctx context.Context,
	st storage.Database,
	blockData blockdata.BlockData,
	policy *isaac.LocalPolicy,
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
) (states.States, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return nil, err
	}

	var conf config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &conf); err != nil {
		return nil, err
	}

	log.Debug().Msg("local is not in suffrage")

	stopped := basicstate.NewStoppedState()
	booting := basicstate.NewBootingState(nodepool.LocalNode(), st, blockData, policy, suffrage)
	joining := basicstate.NewEmptyState()
	consensus := basicstate.NewEmptyState()
	syncing := basicstate.NewSyncingStateNoneSuffrage(st, blockData, policy, nodepool, conf.LocalConfig().SyncInterval())

	return basicstate.NewStates(
		st,
		policy,
		nodepool,
		suffrage,
		nil,
		stopped,
		booting,
		joining,
		consensus,
		syncing,
	)
}
