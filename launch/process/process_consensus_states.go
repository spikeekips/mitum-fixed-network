package process

import (
	"context"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
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
		_ = cs.SetLogger(log)

		return context.WithValue(ctx, ContextValueConsensusStates, cs), nil
	}
}

func processConsensusStates(
	local *isaac.Local,
	pps *prprocessor.Processors,
	suffrage base.Suffrage,
	log logging.Logger,
) (*isaac.ConsensusStates, error) {
	proposalMaker := isaac.NewProposalMaker(local)

	var booting, joining, consensus, syncing, broken isaac.StateHandler
	var err error
	if booting, err = isaac.NewStateBootingHandler(local, suffrage); err != nil {
		return nil, err
	}
	syncing = isaac.NewStateSyncingHandler(local)
	if joining, err = isaac.NewStateJoiningHandler(local, pps); err != nil {
		return nil, err
	}
	if consensus, err = isaac.NewStateConsensusHandler(
		local, pps, suffrage, proposalMaker,
	); err != nil {
		return nil, err
	}
	if broken, err = isaac.NewStateBrokenHandler(local); err != nil {
		return nil, err
	}
	for _, h := range []interface{}{booting, joining, consensus, syncing, broken} {
		if l, ok := h.(logging.SetLogger); ok {
			_ = l.SetLogger(log)
		}
	}

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

	return isaac.NewConsensusStates(
		local, ballotbox, suffrage,
		booting.(*isaac.StateBootingHandler),
		joining.(*isaac.StateJoiningHandler),
		consensus.(*isaac.StateConsensusHandler),
		syncing.(*isaac.StateSyncingHandler),
		broken.(*isaac.StateBrokenHandler),
	)
}
