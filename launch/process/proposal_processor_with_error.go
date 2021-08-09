package process

import (
	"context"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

func NewErrorProcessorNewFunc(
	st storage.Database,
	blockData blockdata.BlockData,
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
	oprHintset *hint.Hintmap,
	whenPreparePoints []config.ErrorPoint,
	whenSavePoints []config.ErrorPoint,
) prprocessor.ProcessorNewFunc {
	return func(proposal ballot.Proposal, initVoteproof base.Voteproof) (prprocessor.Processor, error) {
		pp, err := isaac.NewDefaultProcessor(
			st,
			blockData,
			nodepool,
			suffrage,
			oprHintset,
			proposal,
			initVoteproof,
		)
		if err != nil {
			return nil, err
		}
		return NewErrorProposalProcessor(
			pp,
			whenPreparePoints,
			whenSavePoints,
		), nil
	}
}

type ErrorProposalProcessor struct {
	*isaac.DefaultProcessor
	whenPreparePoints []config.ErrorPoint
	whenSavePoints    []config.ErrorPoint
}

func NewErrorProposalProcessor(
	d *isaac.DefaultProcessor,
	whenPreparePoints, whenSavePoints []config.ErrorPoint,
) *ErrorProposalProcessor {
	d.Logging = logging.NewLogging(func(c zerolog.Context) zerolog.Context {
		return c.Str("module", "error-proposal-processor").
			Int64("height", d.Proposal().Height().Int64()).
			Uint64("round", d.Proposal().Round().Uint64()).
			Stringer("proposal", d.Proposal().Hash())
	})

	return &ErrorProposalProcessor{
		DefaultProcessor:  d,
		whenPreparePoints: whenPreparePoints,
		whenSavePoints:    whenSavePoints,
	}
}

func (pp *ErrorProposalProcessor) Prepare(ctx context.Context) (block.Block, error) {
	if p, found := pp.findPoint(pp.whenPreparePoints); found {
		pp.Log().Debug().Interface("point", p).Msg("prepare-occurring-error")

		if p.Type == config.ErrorTypeWrongBlockHash {
			// NOTE return fake block.Block
			return block.NewBlockV0(
				pp.SuffrageInfo(),
				pp.Proposal().Height(),
				pp.Proposal().Round(),
				pp.Proposal().Hash(),
				pp.BaseManifest().Hash(),
				valuehash.RandomSHA256(),
				valuehash.RandomSHA256(),
				localtime.UTCNow(),
			)
		}
		return nil, errors.Errorf(
			"contest-designed-error: prepare-occurring-error: height=%d round=%d",
			pp.Proposal().Height(),
			pp.Proposal().Round(),
		)
	}

	return pp.DefaultProcessor.Prepare(ctx)
}

func (pp *ErrorProposalProcessor) Save(ctx context.Context) error {
	if p, found := pp.findPoint(pp.whenSavePoints); found {
		pp.Log().Debug().Interface("point", p).Msg("save-occurring-error")

		return errors.Errorf(
			"contest-designed-error: save-occurring-error: height=%d round=%d",
			pp.Proposal().Height(),
			pp.Proposal().Round(),
		)
	}

	return pp.DefaultProcessor.Save(ctx)
}

func (pp *ErrorProposalProcessor) findPoint(points []config.ErrorPoint) (config.ErrorPoint, bool) {
	var found bool
	var point config.ErrorPoint
	for i := range points {
		p := points[i]
		if p.Height == pp.Proposal().Height() && p.Round == pp.Proposal().Round() {
			found = true
			point = p

			break
		}
	}

	return point, found
}
