package process

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
)

func NewErrorProcessorNewFunc(
	local base.Node,
	st storage.Storage,
	blockFS *storage.BlockFS,
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
	oprHintset *hint.Hintmap,
	whenPreparePoints []config.ErrorPoint,
	whenSavePoints []config.ErrorPoint,
) prprocessor.ProcessorNewFunc {
	return func(proposal ballot.Proposal, initVoteproof base.Voteproof) (prprocessor.Processor, error) {
		if pp, err := isaac.NewDefaultProcessor(
			local,
			st,
			blockFS,
			nodepool,
			suffrage,
			oprHintset,
			proposal,
			initVoteproof,
		); err != nil {
			return nil, err
		} else {
			return NewErrorProposalProcessor(
				pp,
				whenPreparePoints,
				whenSavePoints,
			), nil
		}
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
	d.Logging = logging.NewLogging(func(c logging.Context) logging.Emitter {
		return c.Str("module", "error-proposal-processor").
			Hinted("height", d.Proposal().Height()).
			Hinted("round", d.Proposal().Round()).
			Hinted("proposal", d.Proposal().Hash())
	})

	return &ErrorProposalProcessor{
		DefaultProcessor:  d,
		whenPreparePoints: whenPreparePoints,
		whenSavePoints:    whenSavePoints,
	}
}

func (pp *ErrorProposalProcessor) Prepare(ctx context.Context) (block.Block, error) {
	var found bool
	for _, p := range pp.whenPreparePoints {
		if p.Height == pp.Proposal().Height() && p.Round == pp.Proposal().Round() {
			found = true

			break
		}
	}

	if found {
		return nil, xerrors.Errorf(
			"contest-designed-error: prepare-occurring-error: height=%d round=%d",
			pp.Proposal().Height(),
			pp.Proposal().Round(),
		)
	}

	return pp.DefaultProcessor.Prepare(ctx)
}

func (pp *ErrorProposalProcessor) Save(ctx context.Context) error {
	var found bool
	for _, p := range pp.whenSavePoints {
		if p.Height == pp.Proposal().Height() && p.Round == pp.Proposal().Round() {
			found = true

			break
		}
	}

	if found {
		return xerrors.Errorf(
			"contest-designed-error: save-occurring-error: height=%d round=%d",
			pp.Proposal().Height(),
			pp.Proposal().Round(),
		)
	}

	return pp.DefaultProcessor.Save(ctx)
}
