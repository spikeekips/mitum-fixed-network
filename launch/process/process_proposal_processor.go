package process

import (
	"context"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/pm"
)

const ProcessNameProposalProcessor = "proposal_processor"

var ProcessorProposalProcessor pm.Process

func init() {
	if i, err := pm.NewProcess(
		ProcessNameProposalProcessor,
		[]string{
			ProcessNameLocal,
			ProcessNameSuffrage,
		},
		ProcessProposalProcessor,
	); err != nil {
		panic(err)
	} else {
		ProcessorProposalProcessor = i
	}
}

func ProcessProposalProcessor(ctx context.Context) (context.Context, error) {
	var local *isaac.Local
	if err := LoadLocalContextValue(ctx, &local); err != nil {
		return ctx, err
	}

	var suffrage base.Suffrage
	if err := LoadSuffrageContextValue(ctx, &suffrage); err != nil {
		return ctx, err
	}

	pr := isaac.NewDefaultProposalProcessor(local, suffrage)
	if err := pr.Initialize(); err != nil {
		return ctx, err
	}

	return context.WithValue(ctx, ContextValueProposalProcessor, pr), nil
}
