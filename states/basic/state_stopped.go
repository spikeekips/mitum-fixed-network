package basicstates

import (
	"github.com/spikeekips/mitum/base"
)

type StoppedState struct {
	*BaseState
}

func NewStoppedState() *StoppedState {
	return &StoppedState{
		BaseState: NewBaseState(base.StateStopped),
	}
}
