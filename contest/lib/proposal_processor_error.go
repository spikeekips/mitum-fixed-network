package contestlib

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/storage"
)

type ErrorProposalProcessor struct {
	*isaac.ProposalProcessorV0
	initPoints   []BlockPoint
	acceptPoints []BlockPoint
}

func NewErrorProposalProcessor(
	localstate *isaac.Localstate,
	suffrage base.Suffrage,
	initPoints []BlockPoint,
	acceptPoints []BlockPoint,
) *ErrorProposalProcessor {
	return &ErrorProposalProcessor{
		ProposalProcessorV0: isaac.NewProposalProcessorV0(localstate, suffrage),
		initPoints:          initPoints,
		acceptPoints:        acceptPoints,
	}
}

func (ep *ErrorProposalProcessor) ProcessINIT(ph valuehash.Hash, initVoteproof base.Voteproof) (
	block.Block, error,
) {
	for _, h := range ep.initPoints {
		if h.Height == initVoteproof.Height() && h.Round == initVoteproof.Round() {
			return nil, xerrors.Errorf("contest-designed-error")
		}
	}

	return ep.ProposalProcessorV0.ProcessINIT(ph, initVoteproof)
}

func (ep *ErrorProposalProcessor) ProcessACCEPT(ph valuehash.Hash, acceptVoteproof base.Voteproof) (
	storage.BlockStorage, error,
) {
	for _, h := range ep.acceptPoints {
		if h.Height == acceptVoteproof.Height() && h.Round == acceptVoteproof.Round() {
			return nil, xerrors.Errorf("contest-designed-error")
		}
	}

	return ep.ProposalProcessorV0.ProcessACCEPT(ph, acceptVoteproof)
}
