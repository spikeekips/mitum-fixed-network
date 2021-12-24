package process

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/network/discovery/memberlist"
	"github.com/spikeekips/mitum/states"
	basicstate "github.com/spikeekips/mitum/states/basic"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
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
			ProcessNameBlockdata,
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

	var db storage.Database
	if err := LoadDatabaseContextValue(ctx, &db); err != nil {
		return ctx, err
	}

	var bd blockdata.Blockdata
	if err := LoadBlockdataContextValue(ctx, &bd); err != nil {
		return ctx, err
	}

	var suffrage base.Suffrage
	if err := LoadSuffrageContextValue(ctx, &suffrage); err != nil {
		return ctx, err
	}

	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	cs, err := processConsensusStates(ctx, db, bd, policy, nodepool, suffrage)
	if err != nil {
		return ctx, err
	}

	if i, ok := cs.(logging.SetLogging); ok {
		_ = i.SetLogging(log)
	}

	return context.WithValue(ctx, ContextValueConsensusStates, cs), nil
}

func processConsensusStates(
	ctx context.Context,
	db storage.Database,
	bd blockdata.Blockdata,
	policy *isaac.LocalPolicy,
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
) (states.States, error) {
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return nil, err
	}

	var pps *prprocessor.Processors
	if err := LoadProposalProcessorContextValue(ctx, &pps); err != nil {
		if !errors.Is(err, util.ContextValueNotFoundError) {
			return nil, err
		}
	}

	proposalMaker := isaac.NewProposalMaker(nodepool.LocalNode(), db, policy)

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
	_ = ballotbox.SetLogging(log)

	joiner, err := createDiscoveryJoiner(ctx, nodepool, suffrage)
	if err != nil {
		return nil, err
	}

	hd, err := createHandover(ctx, policy, nodepool, suffrage)
	if err != nil {
		return nil, err
	}

	stopped := basicstate.NewStoppedState()
	booting := basicstate.NewBootingState(nodepool.LocalNode(), db, bd, policy, suffrage)
	joining := basicstate.NewJoiningState(nodepool.LocalNode(), db, policy, suffrage, ballotbox)
	consensus := basicstate.NewConsensusState(db, policy, nodepool, suffrage, proposalMaker, pps)
	syncing := basicstate.NewSyncingState(db, bd, policy, nodepool, suffrage)
	handover := basicstate.NewHandoverState(db, policy, nodepool, suffrage, pps)

	return basicstate.NewStates(
		db,
		policy,
		nodepool,
		suffrage,
		ballotbox,
		stopped,
		booting,
		joining,
		consensus,
		syncing,
		handover,
		joiner,
		hd,
	)
}

func createDiscoveryJoiner(
	ctx context.Context,
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
) (*states.DiscoveryJoiner, error) {
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return nil, err
	}

	var encs *encoder.Encoders
	if err := config.LoadEncodersContextValue(ctx, &encs); err != nil {
		return nil, err
	}

	var dis *memberlist.Discovery
	if err := util.LoadFromContextValue(ctx, ContextValueDiscovery, &dis); err != nil {
		if errors.Is(err, util.ContextValueNotFoundError) {
			return nil, nil
		}

		log.Log().Debug().Err(err).Msgf("discovery joiner disabled for %T", dis)

		return nil, nil
	}

	var cis []network.ConnInfo
	if err := LoadDiscoveryConnInfosContextValue(ctx, &cis); err != nil {
		if !errors.Is(err, util.ContextValueNotFoundError) {
			return nil, err
		}
	}

	if len(cis) < 1 {
		return nil, nil
	}

	joiner, err := states.NewDiscoveryJoiner(nodepool, suffrage, dis, cis)
	if err != nil {
		return nil, fmt.Errorf("failed to make *DiscoveryJoiner: %w", err)
	}

	_ = joiner.SetLogging(log)

	return joiner, nil
}

func createHandover(
	ctx context.Context,
	policy *isaac.LocalPolicy,
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
) (*basicstate.Handover, error) {
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return nil, err
	}

	var encs *encoder.Encoders
	if err := config.LoadEncodersContextValue(ctx, &encs); err != nil {
		return nil, err
	}

	var ln config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &ln); err != nil {
		return nil, err
	}
	connInfo := ln.Network().ConnInfo()

	var cis []network.ConnInfo
	if err := LoadDiscoveryConnInfosContextValue(ctx, &cis); err != nil {
		if !errors.Is(err, util.ContextValueNotFoundError) {
			return nil, err
		}
	}

	handover, err := basicstate.NewHandoverWithDiscoveryURL(connInfo, encs, policy, nodepool, suffrage, cis)
	if err != nil {
		return nil, fmt.Errorf("failed to make Handover: %w", err)
	}
	_ = handover.SetLogging(log)

	return handover, nil
}
