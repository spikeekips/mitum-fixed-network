package process

import (
	"context"

	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/pm"
	"golang.org/x/xerrors"
)

const ProcessNameStartConsensusStates = "start_consensus_states"

var ProcessorStartConsensusStates pm.Process

func init() {
	if i, err := pm.NewProcess(
		ProcessNameStartConsensusStates,
		[]string{
			ProcessNameConsensusStates,
		},
		ProcessStartConsensusStates,
	); err != nil {
		panic(err)
	} else {
		ProcessorStartConsensusStates = i
	}
}

func ProcessStartConsensusStates(ctx context.Context) (context.Context, error) {
	var pps *prprocessor.Processors
	if err := LoadProposalProcessorContextValue(ctx, &pps); err != nil {
		return ctx, err
	}

	var cs *isaac.ConsensusStates
	if err := LoadConsensusStatesContextValue(ctx, &cs); err != nil {
		return ctx, err
	}

	if err := pps.Start(); err != nil {
		return ctx, xerrors.Errorf("failed to start Processors: %w", err)
	} else if err := cs.Start(); err != nil {
		return ctx, xerrors.Errorf("failed to start consensus states: %w", err)
	}

	return ctx, nil
}
