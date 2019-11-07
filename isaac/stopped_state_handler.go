package isaac

import (
	"sync"

	"github.com/rs/zerolog"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/node"
)

type StoppedStateHandler struct {
	sync.RWMutex
	*common.Logger
	started bool
}

func NewStoppedStateHandler() *StoppedStateHandler {
	return &StoppedStateHandler{
		Logger: common.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "stopped-state-handler")
		}),
	}
}

func (ss *StoppedStateHandler) Start() error {
	ss.Lock()
	defer ss.Unlock()

	ss.started = true

	return nil
}

func (ss *StoppedStateHandler) Stop() error {
	ss.Lock()
	defer ss.Unlock()

	ss.started = false

	return nil
}

func (ss *StoppedStateHandler) IsStopped() bool {
	ss.RLock()
	defer ss.RUnlock()

	return ss.started
}

func (ss *StoppedStateHandler) Activate(StateContext) error {
	return nil
}

func (ss *StoppedStateHandler) Deactivate() error {
	return nil
}

func (ss *StoppedStateHandler) SetChanState(chan StateContext) StateHandler {
	return ss
}

func (ss *StoppedStateHandler) State() node.State {
	return node.StateStopped
}

func (ss *StoppedStateHandler) ReceiveProposal(Proposal) error {
	return nil
}

func (ss *StoppedStateHandler) ReceiveVoteResult(VoteResult) error {
	return nil
}
