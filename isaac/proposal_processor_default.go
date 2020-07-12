package isaac

import (
	"sync"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

type DefaultProposalProcessor struct {
	sync.RWMutex
	*logging.Logging
	localstate *Localstate
	suffrage   base.Suffrage
	pp         *internalDefaultProposalProcessor
}

func NewDefaultProposalProcessor(localstate *Localstate, suffrage base.Suffrage) *DefaultProposalProcessor {
	return &DefaultProposalProcessor{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "default-proposal-processor")
		}),
		localstate: localstate,
		suffrage:   suffrage,
	}
}

func (dp *DefaultProposalProcessor) Initialize() error {
	return nil
}

func (dp *DefaultProposalProcessor) IsProcessed(ph valuehash.Hash) bool {
	dp.RLock()
	defer dp.RUnlock()

	if dp.pp == nil {
		return false
	}

	return dp.pp.proposal.Hash().Equal(ph)
}

func (dp *DefaultProposalProcessor) processed(ph valuehash.Hash) *internalDefaultProposalProcessor {
	dp.RLock()
	defer dp.RUnlock()

	if dp.pp == nil {
		return nil
	}

	if !dp.pp.proposal.Hash().Equal(ph) {
		return nil
	}

	return dp.pp
}

func (dp *DefaultProposalProcessor) setProcessor(pp *internalDefaultProposalProcessor) {
	dp.Lock()
	defer dp.Unlock()

	dp.pp = pp
}

func (dp *DefaultProposalProcessor) ProcessINIT(ph valuehash.Hash, initVoteproof base.Voteproof) (block.Block, error) {
	if pp := dp.processed(ph); pp != nil {
		return pp.block, nil
	}

	if initVoteproof.Stage() != base.StageINIT {
		return nil, xerrors.Errorf("ProcessINIT needs INIT Voteproof")
	}

	var proposal ballot.Proposal
	if pr, err := dp.checkProposal(ph, initVoteproof); err != nil {
		return nil, err
	} else {
		proposal = pr
	}

	pp, err := newInternalDefaultProposalProcessor(dp.localstate, dp.suffrage, proposal)
	if err != nil {
		return nil, err
	}

	_ = pp.SetLogger(dp.Log())

	dp.setProcessor(nil)

	blk, err := pp.processINIT(initVoteproof)
	if err != nil {
		return nil, err
	}

	dp.setProcessor(pp)

	return blk, nil
}

func (dp *DefaultProposalProcessor) ProcessACCEPT(
	ph valuehash.Hash, acceptVoteproof base.Voteproof,
) (storage.BlockStorage, error) {
	if acceptVoteproof.Stage() != base.StageACCEPT {
		return nil, xerrors.Errorf("Processaccept needs ACCEPT Voteproof")
	}

	pp := dp.processed(ph)
	if pp == nil {
		return nil, xerrors.Errorf("not processed ProcessINIT")
	}

	if err := pp.setACCEPTVoteproof(acceptVoteproof); err != nil {
		return nil, err
	}

	dp.setProcessor(nil)

	return pp.bs, nil
}

func (dp *DefaultProposalProcessor) checkProposal(
	ph valuehash.Hash, initVoteproof base.Voteproof,
) (ballot.Proposal, error) {
	var proposal ballot.Proposal
	if sl, found, err := dp.localstate.Storage().Seal(ph); !found {
		return nil, storage.NotFoundError.Errorf("seal not found")
	} else if err != nil {
		return nil, err
	} else if pr, ok := sl.(ballot.Proposal); !ok {
		return nil, xerrors.Errorf("seal is not Proposal: %T", sl)
	} else {
		proposal = pr
	}

	timespan := dp.localstate.Policy().TimespanValidBallot()
	if proposal.SignedAt().Before(initVoteproof.FinishedAt().Add(timespan * -1)) {
		return nil, xerrors.Errorf(
			"Proposal was sent before Voteproof; SignedAt=%s now=%s timespan=%s",
			proposal.SignedAt(), initVoteproof.FinishedAt(), timespan,
		)
	}

	return proposal, nil
}
