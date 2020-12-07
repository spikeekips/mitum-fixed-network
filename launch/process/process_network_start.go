package process

import (
	"context"

	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/network"
	"golang.org/x/xerrors"
)

const ProcessNameStartNetwork = "start_network"

var ProcessorStartNetwork pm.Process

func init() {
	if i, err := pm.NewProcess(
		ProcessNameStartNetwork,
		[]string{
			ProcessNameNetwork,
		},
		ProcessStartQuicNetwork,
	); err != nil {
		panic(err)
	} else {
		ProcessorStartNetwork = i
	}
}

func ProcessStartQuicNetwork(ctx context.Context) (context.Context, error) {
	var nt network.Server
	if err := LoadNetworkContextValue(ctx, &nt); err != nil {
		return ctx, err
	}

	if err := nt.Start(); err != nil {
		return ctx, xerrors.Errorf("failed to start network: %w", err)
	}

	return ctx, nil
}
