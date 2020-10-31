package launcher

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/valuehash"
)

type ErrorProposalProcessor struct {
	*isaac.DefaultProposalProcessor
	initPoints   []BlockPoint
	acceptPoints []BlockPoint
}

func NewErrorProposalProcessor(
	local *isaac.Local,
	suffrage base.Suffrage,
	initPoints []BlockPoint,
	acceptPoints []BlockPoint,
) *ErrorProposalProcessor {
	return &ErrorProposalProcessor{
		DefaultProposalProcessor: isaac.NewDefaultProposalProcessor(local, suffrage),
		initPoints:               initPoints,
		acceptPoints:             acceptPoints,
	}
}

func (ep *ErrorProposalProcessor) ProcessINIT(ph valuehash.Hash, initVoteproof base.Voteproof) (
	block.Block, error,
) {
	for _, h := range ep.initPoints {
		if h.Height == initVoteproof.Height() && h.Round == initVoteproof.Round() {
			return nil, xerrors.Errorf("contest-designed-error: occurring-error")
		}
	}

	return ep.DefaultProposalProcessor.ProcessINIT(ph, initVoteproof)
}

func (ep *ErrorProposalProcessor) ProcessACCEPT(ph valuehash.Hash, acceptVoteproof base.Voteproof) (
	storage.BlockStorage, error,
) {
	for _, h := range ep.acceptPoints {
		if h.Height == acceptVoteproof.Height() && h.Round == acceptVoteproof.Round() {
			return nil, xerrors.Errorf("contest-designed-error: occurring-error")
		}
	}

	return ep.DefaultProposalProcessor.ProcessACCEPT(ph, acceptVoteproof)
}
