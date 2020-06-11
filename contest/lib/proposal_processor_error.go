package contestlib

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/storage"
	"golang.org/x/xerrors"
)

type ErrorProposalProcessor struct {
	*isaac.ProposalProcessorV0
	initHeights   []base.Height
	acceptHeights []base.Height
}

func NewErrorProposalProcessor(
	localstate *isaac.Localstate,
	suffrage base.Suffrage,
	initHeights []base.Height,
	acceptHeights []base.Height,
) *ErrorProposalProcessor {
	return &ErrorProposalProcessor{
		ProposalProcessorV0: isaac.NewProposalProcessorV0(localstate, suffrage),
		initHeights:         initHeights,
		acceptHeights:       acceptHeights,
	}
}

func (ep *ErrorProposalProcessor) ProcessINIT(ph valuehash.Hash, initVoteproof base.Voteproof) (
	block.Block, error,
) {
	for _, h := range ep.initHeights {
		if h == initVoteproof.Height() {
			return nil, xerrors.Errorf("contest-designed-error")
		}
	}

	return ep.ProposalProcessorV0.ProcessINIT(ph, initVoteproof)
}

func (ep *ErrorProposalProcessor) ProcessACCEPT(ph valuehash.Hash, acceptVoteproof base.Voteproof) (
	storage.BlockStorage, error,
) {
	for _, h := range ep.acceptHeights {
		if h == acceptVoteproof.Height() {
			return nil, xerrors.Errorf("contest-designed-error")
		}
	}

	return ep.ProposalProcessorV0.ProcessACCEPT(ph, acceptVoteproof)
}
