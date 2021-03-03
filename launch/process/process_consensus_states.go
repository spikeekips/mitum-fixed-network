package process

import (
	"context"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/states"
	basicstate "github.com/spikeekips/mitum/states/basic"
	"github.com/spikeekips/mitum/util/logging"
)

const ProcessNameConsensusStates = "consensus_states"

var ProcessorConsensusStates pm.Process

func init() {
	if i, err := pm.NewProcess(
		ProcessNameConsensusStates,
		[]string{
			ProcessNameLocal,
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
	var local *isaac.Local
	if err := LoadLocalContextValue(ctx, &local); err != nil {
		return ctx, err
	}

	var pps *prprocessor.Processors
	if err := LoadProposalProcessorContextValue(ctx, &pps); err != nil {
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

	if cs, err := processConsensusStates(local, pps, suffrage, log); err != nil {
		return ctx, err
	} else {
		if i, ok := cs.(logging.SetLogger); ok {
			_ = i.SetLogger(log)
		}

		return context.WithValue(ctx, ContextValueConsensusStates, cs), nil
	}
}

func processConsensusStates(
	local *isaac.Local,
	pps *prprocessor.Processors,
	suffrage base.Suffrage,
	log logging.Logger,
) (states.States, error) {
	ballotbox := isaac.NewBallotbox(
		suffrage.Nodes,
		func() base.Threshold {
			if t, err := base.NewThreshold(
				uint(len(suffrage.Nodes())),
				local.Policy().ThresholdRatio(),
			); err != nil {
				panic(err)
			} else {
				return t
			}
		},
	)
	_ = ballotbox.SetLogger(log)

	proposalMaker := isaac.NewProposalMaker(local)

	stopped := basicstate.NewStoppedState()
	booting := basicstate.NewBootingState(local.Storage(), local.BlockFS(), local.Policy(), suffrage)
	joining := basicstate.NewJoiningState(local.Node(), local.Storage(), local.BlockFS(), local.Policy(),
		suffrage, ballotbox)
	consensus := basicstate.NewConsensusState(
		local.Node(), local.Storage(), local.BlockFS(), local.Policy(), local.Nodes(), suffrage, proposalMaker, pps)
	syncing := basicstate.NewSyncingState(local.Node(), local.Storage(), local.BlockFS(), local.Policy(), local.Nodes())

	return basicstate.NewStates(
		local.Node(),
		local.Storage(),
		local.BlockFS(),
		local.Policy(),
		local.Nodes(),
		suffrage,
		ballotbox,
		stopped,
		booting,
		joining,
		consensus,
		syncing,
	)
}
