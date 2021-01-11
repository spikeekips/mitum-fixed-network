package process

import (
	"context"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
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
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var l config.LocalNode
	var conf config.ProposalProcessor
	if err := config.LoadConfigContextValue(ctx, &l); err != nil {
		return ctx, err
	} else {
		conf = l.ProposalProcessor()
	}

	var newFunc prprocessor.ProcessorNewFunc
	switch t := conf.(type) {
	case config.ErrorProposalProcessor:
		log.Debug().Interface("conf", conf).Msg("ErrorProcessor will be used")

		if i, err := processErrorProposalProcessor(ctx, t); err != nil {
			return ctx, err
		} else {
			newFunc = i
		}
	default:
		log.Debug().Interface("conf", conf).Msg("DefaultProcessor will be used")

		if i, err := processDefaultProposalProcessor(ctx); err != nil {
			return ctx, err
		} else {
			newFunc = i
		}
	}

	pps := prprocessor.NewProcessors(newFunc, nil)
	if err := pps.Initialize(); err != nil {
		return ctx, err
	}

	_ = pps.SetLogger(log)

	return context.WithValue(ctx, ContextValueProposalProcessor, pps), nil
}

func processDefaultProposalProcessor(ctx context.Context) (prprocessor.ProcessorNewFunc, error) {
	var local *isaac.Local
	if err := LoadLocalContextValue(ctx, &local); err != nil {
		return nil, err
	}

	var sf storage.Storage
	if err := LoadStorageContextValue(ctx, &sf); err != nil {
		return nil, err
	}

	var blockFS *storage.BlockFS
	if err := LoadBlockFSContextValue(ctx, &blockFS); err != nil {
		return nil, err
	}

	var suffrage base.Suffrage
	if err := LoadSuffrageContextValue(ctx, &suffrage); err != nil {
		return nil, err
	}

	var oprs *hint.Hintmap
	if err := LoadOperationProcessorsContextValue(ctx, &oprs); err != nil {
		return nil, err
	}

	return isaac.NewDefaultProcessorNewFunc(
		local.Node(),
		sf,
		blockFS,
		local.Nodes(),
		suffrage,
		oprs,
	), nil
}

func processErrorProposalProcessor(
	ctx context.Context,
	conf config.ErrorProposalProcessor,
) (prprocessor.ProcessorNewFunc, error) {
	var l logging.Logger
	if err := config.LoadLogContextValue(ctx, &l); err != nil {
		return nil, err
	}

	if len(conf.WhenPreparePoints) < 1 && len(conf.WhenSavePoints) < 1 {
		l.Debug().Msg("ErrorProposalProcessor was given, but block points are empty. DefaultProposalProcessor will be used")

		return processDefaultProposalProcessor(ctx)
	}

	var local *isaac.Local
	if err := LoadLocalContextValue(ctx, &local); err != nil {
		return nil, err
	}

	var sf storage.Storage
	if err := LoadStorageContextValue(ctx, &sf); err != nil {
		return nil, err
	}

	var blockFS *storage.BlockFS
	if err := LoadBlockFSContextValue(ctx, &blockFS); err != nil {
		return nil, err
	}

	var suffrage base.Suffrage
	if err := LoadSuffrageContextValue(ctx, &suffrage); err != nil {
		return nil, err
	}

	var oprs *hint.Hintmap
	if err := LoadOperationProcessorsContextValue(ctx, &oprs); err != nil {
		return nil, err
	}

	return NewErrorProcessorNewFunc(
		local.Node(),
		sf,
		blockFS,
		local.Nodes(),
		suffrage,
		oprs,
		conf.WhenPreparePoints,
		conf.WhenSavePoints,
	), nil
}
