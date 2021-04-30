package basicstates

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
)

type EmptyState struct{}

func NewEmptyState() *EmptyState {
	return &EmptyState{}
}

func (st *EmptyState) Enter(StateSwitchContext) (func() error, error) {
	return nil, nil
}

func (st *EmptyState) Exit(StateSwitchContext) (func() error, error) {
	return nil, nil
}

func (st *EmptyState) ProcessProposal(ballot.Proposal) error {
	return nil
}

func (st *EmptyState) ProcessVoteproof(base.Voteproof) error {
	return nil
}

func (st *EmptyState) SetStates(*States) State {
	return nil
}
