package isaac

import (
	"sync"

	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/valuehash"
)

type ProposalProcessor interface {
	ProcessINIT(valuehash.Hash /* Proposal.Hash() */, Voteproof /* INIT Voteproof */, []byte) (Block, error)
	ProcessACCEPT(valuehash.Hash /* Proposal.Hash() */, Voteproof /* ACCEPT Voteproof */, []byte) (BlockStorage, error)
}

type ProposalProcessorV0 struct {
	*logging.Logger
	localstate *Localstate
	blocks     *sync.Map
}

func NewProposalProcessorV0(localstate *Localstate) *ProposalProcessorV0 {
	return &ProposalProcessorV0{
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "proposal-processor-v0")
		}),
		localstate: localstate,
		blocks:     &sync.Map{},
	}
}

func (dp *ProposalProcessorV0) ProcessINIT(ph valuehash.Hash, initVoteproof Voteproof, b []byte) (Block, error) {
	if i, found := dp.blocks.Load(ph); found {
		return i.(Block), nil
	}

	var proposal Proposal
	if sl, err := dp.localstate.Storage().Seal(ph); err != nil {
		return nil, err
	} else if pr, ok := sl.(Proposal); !ok {
		return nil, xerrors.Errorf("seal is not Proposal: %T", sl)
	} else {
		proposal = pr
	}

	if proposal.Height() != Height(0) { // check proposed time if not genesis proposal
		ivp := dp.localstate.LastINITVoteproof()
		if ivp == nil {
			return nil, xerrors.Errorf("last INIT Voteproof is missing")
		}

		timespan := dp.localstate.Policy().TimespanValidBallot()
		if proposal.SignedAt().Before(ivp.FinishedAt().Add(timespan * -1)) {
			return nil, xerrors.Errorf(
				"Proposal was sent before Voteproof; SignedAt=%s now=%s timespan=%s",
				proposal.SignedAt(), ivp.FinishedAt(), timespan,
			)
		}
	}

	lastBlock := dp.localstate.LastBlock()
	if lastBlock == nil {
		return nil, xerrors.Errorf("last block is empty")
	}

	blockOperations, err := dp.processSeals(proposal, b)
	if err != nil {
		return nil, err
	}

	blockStates, err := dp.processStates(proposal, b)
	if err != nil {
		return nil, err
	}

	var block Block
	if b, err := NewBlockV0(
		proposal.Height(), proposal.Round(), proposal.Hash(), lastBlock.Hash(),
		blockOperations, blockStates,
		b,
	); err != nil {
		return nil, err
	} else {
		block = b.SetINITVoteproof(initVoteproof)
	}

	dp.blocks.Store(ph, block)

	return block, nil
}

// TODO b is NetworkID
func (dp *ProposalProcessorV0) ProcessACCEPT(
	ph valuehash.Hash, acceptVoteproof Voteproof, _ []byte,
) (BlockStorage, error) {
	var block Block
	if i, found := dp.blocks.Load(ph); !found {
		return nil, xerrors.Errorf("not processed ProcessINIT")
	} else {
		block = i.(Block).SetACCEPTVoteproof(acceptVoteproof)
	}

	return dp.localstate.Storage().OpenBlockStorage(block)
}

func (dp *ProposalProcessorV0) processSeals(Proposal, []byte) (valuehash.Hash, error) {
	return nil, nil
}

func (dp *ProposalProcessorV0) processStates(Proposal, []byte) (valuehash.Hash, error) {
	return nil, nil
}
