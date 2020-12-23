package process

import (
	"context"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/hint"
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

	var sf storage.Storage
	if err := LoadStorageContextValue(ctx, &sf); err != nil {
		return ctx, err
	}

	var blockFS *storage.BlockFS
	if err := LoadBlockFSContextValue(ctx, &blockFS); err != nil {
		return ctx, err
	}

	var suffrage base.Suffrage
	if err := LoadSuffrageContextValue(ctx, &suffrage); err != nil {
		return ctx, err
	}

	var oprs *hint.Hintmap
	if err := LoadOperationProcessorsContextValue(ctx, &oprs); err != nil {
		return ctx, err
	}

	pps := prprocessor.NewProcessors(
		isaac.NewDefaultProcessorNewFunc(
			local.Node(),
			sf,
			blockFS,
			local.Nodes(),
			suffrage,
			oprs,
		),
		nil,
	)
	if err := pps.Initialize(); err != nil {
		return ctx, err
	}

	return context.WithValue(ctx, ContextValueProposalProcessor, pps), nil
}
