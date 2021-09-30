package basicstates

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/localtime"
)

var BlockPrefixFailedProcessProposal = []byte("processproposalfailed")

type ConsensusState struct {
	*BaseConsensusState
}

func NewConsensusState(
	db storage.Database,
	policy *isaac.LocalPolicy,
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
	proposalMaker *isaac.ProposalMaker,
	pps *prprocessor.Processors,
) *ConsensusState {
	bc := NewBaseConsensusState(base.StateConsensus,
		db, policy, nodepool, suffrage, proposalMaker, pps)

	st := &ConsensusState{
		BaseConsensusState: bc,
	}

	bc.broadcastACCEPTBallot = st.defaultBroadcastACCEPTBallot
	bc.broadcastNewINITBallot = st.defaultBroadcastNewINITBallot
	bc.prepareProposal = bc.defaultPrepareProposal

	return st
}

func (st *ConsensusState) Enter(sctx StateSwitchContext) (func() error, error) {
	if st.underHandover() {
		return nil, errors.Errorf("consensus should not be entered under handover")
	}

	return st.BaseConsensusState.Enter(sctx)
}

func (st *ConsensusState) Exit(sctx StateSwitchContext) (func() error, error) {
	callback, err := st.BaseConsensusState.Exit(sctx)
	if err != nil {
		return nil, err
	}

	return func() error {
		if err := callback(); err != nil {
			return err
		}

		return st.Timers().StopTimers([]localtime.TimerID{
			TimerIDBroadcastINITBallot,
			TimerIDBroadcastProposal,
			TimerIDFindProposal,
		})
	}, nil
}
