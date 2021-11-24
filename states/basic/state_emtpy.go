package basicstates

import (
	"github.com/spikeekips/mitum/base"
)

type EmptyState struct{}

func NewEmptyState() *EmptyState {
	return &EmptyState{}
}

func (*EmptyState) Enter(sctx StateSwitchContext) (func() error, error) {
	return nil, sctx.Return()
}

func (*EmptyState) Exit(StateSwitchContext) (func() error, error) {
	return nil, nil
}

func (*EmptyState) ProcessProposal(base.Proposal) error {
	return nil
}

func (*EmptyState) ProcessVoteproof(base.Voteproof) error {
	return nil
}

func (*EmptyState) SetStates(*States) State {
	return nil
}
