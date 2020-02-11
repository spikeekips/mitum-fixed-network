package isaac

import (
	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/valuehash"
)

type ProposalProcessor interface {
	Process(valuehash.Hash /* Proposal.Hash() */, []byte) (Block, error)
}

type ProposalProcessorV0 struct {
	*logging.Logger
	localState  *LocalState
	sealStorage SealStorage
}

func NewProposalProcessorV0(localState *LocalState, sealStorage SealStorage) ProposalProcessorV0 {
	return ProposalProcessorV0{
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "proposal-processor-v0")
		}),
		localState:  localState,
		sealStorage: sealStorage,
	}
}

// TODO b is NetworkID
func (dp ProposalProcessorV0) Process(ph valuehash.Hash, b []byte) (Block, error) {
	var proposal Proposal
	if sl, found, err := dp.sealStorage.Seal(ph); err != nil || !found {
		if err != nil {
			return nil, err
		}

		return nil, xerrors.Errorf("Proposal not found; proposal=%s", ph)
	} else {
		proposal = sl.(Proposal)
	}

	lastBlock := dp.localState.LastBlock()
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

	ivp := dp.localState.LastINITVoteProof()
	if ivp == nil {
		return nil, xerrors.Errorf("last INIT VoteProof is missing")
	}

	{ // check proposed time
		timespan := dp.localState.Policy().TimespanValidBallot()
		if proposal.SignedAt().Before(ivp.FinishedAt().Add(timespan * -1)) {
			return nil, xerrors.Errorf(
				"Proposal was sent before VoteProof; SignedAt=%s now=%s timespan=%s",
				proposal.SignedAt(), ivp.FinishedAt(), timespan,
			)
		}
	}

	avp := dp.localState.LastACCEPTVoteProof()
	if ivp == nil {
		return nil, xerrors.Errorf("last ACCEPT VoteProof is missing")
	}

	block, err := NewBlockV0(
		proposal.Height(), proposal.Round(), proposal.Hash(), lastBlock.Hash(),
		blockOperations, blockStates, ivp, avp, b,
	)
	if err != nil {
		return nil, err
	}

	return block, nil
}

func (dp ProposalProcessorV0) processSeals(Proposal, []byte) (valuehash.Hash, error) {
	return nil, nil
}

func (dp ProposalProcessorV0) processStates(Proposal, []byte) (valuehash.Hash, error) {
	return nil, nil
}
