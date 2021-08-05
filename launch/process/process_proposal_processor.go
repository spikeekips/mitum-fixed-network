package process

import (
	"context"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
)

const ProcessNameProposalProcessor = "proposal_processor"

var ProcessorProposalProcessor pm.Process

func init() {
	if i, err := pm.NewProcess(
		ProcessNameProposalProcessor,
		[]string{
			ProcessNameLocalNode,
			ProcessNameDatabase,
			ProcessNameBlockData,
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
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var nodepool *network.Nodepool
	if err := LoadNodepoolContextValue(ctx, &nodepool); err != nil {
		return nil, err
	}

	var suffrage base.Suffrage
	if err := LoadSuffrageContextValue(ctx, &suffrage); err != nil {
		return nil, err
	}

	if !suffrage.IsInside(nodepool.LocalNode().Address()) {
		log.Log().Debug().Msg("none-suffrage node; proposal processor will not be used")

		return ctx, nil
	}

	var l config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &l); err != nil {
		return ctx, err
	}
	conf := l.ProposalProcessor()

	var newFunc prprocessor.ProcessorNewFunc
	switch t := conf.(type) {
	case config.ErrorProposalProcessor:
		log.Log().Debug().Interface("conf", conf).Msg("ErrorProcessor will be used")

		i, err := processErrorProposalProcessor(ctx, t)
		if err != nil {
			return ctx, err
		}
		newFunc = i
	default:
		log.Log().Debug().Interface("conf", conf).Msg("DefaultProcessor will be used")

		i, err := processDefaultProposalProcessor(ctx)
		if err != nil {
			return ctx, err
		}
		newFunc = i
	}

	pps := prprocessor.NewProcessors(newFunc, nil)
	if err := pps.Initialize(); err != nil {
		return ctx, err
	}

	_ = pps.SetLogging(log)

	return context.WithValue(ctx, ContextValueProposalProcessor, pps), nil
}

func processDefaultProposalProcessor(ctx context.Context) (prprocessor.ProcessorNewFunc, error) {
	var nodepool *network.Nodepool
	if err := LoadNodepoolContextValue(ctx, &nodepool); err != nil {
		return nil, err
	}

	var sf storage.Database
	if err := LoadDatabaseContextValue(ctx, &sf); err != nil {
		return nil, err
	}

	var blockData blockdata.BlockData
	if err := LoadBlockDataContextValue(ctx, &blockData); err != nil {
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
		sf,
		blockData,
		nodepool,
		suffrage,
		oprs,
	), nil
}

func processErrorProposalProcessor(
	ctx context.Context,
	conf config.ErrorProposalProcessor,
) (prprocessor.ProcessorNewFunc, error) {
	var l *logging.Logging
	if err := config.LoadLogContextValue(ctx, &l); err != nil {
		return nil, err
	}

	if len(conf.WhenPreparePoints) < 1 && len(conf.WhenSavePoints) < 1 {
		l.Log().Debug().Msg("ErrorProposalProcessor was given, but block points are empty. DefaultProposalProcessor will be used") // revive:disable-line:line-length-limit

		return processDefaultProposalProcessor(ctx)
	}

	var nodepool *network.Nodepool
	if err := LoadNodepoolContextValue(ctx, &nodepool); err != nil {
		return nil, err
	}

	var sf storage.Database
	if err := LoadDatabaseContextValue(ctx, &sf); err != nil {
		return nil, err
	}

	var blockData blockdata.BlockData
	if err := LoadBlockDataContextValue(ctx, &blockData); err != nil {
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
		sf,
		blockData,
		nodepool,
		suffrage,
		oprs,
		conf.WhenPreparePoints,
		conf.WhenSavePoints,
	), nil
}
