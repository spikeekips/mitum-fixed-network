package isaac

import (
	"context"
	"sync"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/node"
)

type BootingStateHandler struct {
	sync.RWMutex
	*common.ZLogger
	started         bool
	chanState       chan StateContext
	proposalChecker *common.ChainChecker
}

func NewBootingStateHandler(homeState *HomeState) *BootingStateHandler {
	return &BootingStateHandler{
		ZLogger: common.NewZLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "booting-state-handler")
		}),
		proposalChecker: NewProposalCheckerBooting(homeState),
	}
}

func (bs *BootingStateHandler) Start() error {
	// TODO health check:
	//	- blocks in storage is valid
	//	- network status
	//	- connectivity of suffrage members
	//	- etc

	// NOTE everything ok, move to next state, JOIN

	bs.Lock()
	defer bs.Unlock()
	bs.started = true

	return nil
}

func (bs *BootingStateHandler) Stop() error {
	bs.Lock()
	defer bs.Unlock()
	bs.started = false

	return nil
}

func (bs *BootingStateHandler) IsStopped() bool {
	bs.RLock()
	defer bs.RUnlock()
	return bs.started
}

func (bs *BootingStateHandler) Activate(StateContext) error {
	go func() {
		bs.chanState <- NewStateContext(node.StateJoin)
	}()

	return nil
}

func (bs *BootingStateHandler) Deactivate() error {
	return nil
}

func (bs *BootingStateHandler) SetChanState(ch chan StateContext) StateHandler {
	bs.chanState = ch
	return bs
}

func (bs *BootingStateHandler) State() node.State {
	return node.StateBooting
}

func (bs *BootingStateHandler) ReceiveProposal(proposal Proposal) error {
	err := bs.proposalChecker.
		New(context.TODO()).
		SetContext(
			"proposal", proposal,
		).
		Check()

	if err != nil {
		return err
	}

	return nil
}

func (bs *BootingStateHandler) ReceiveVoteResult(VoteResult) error {
	return nil
}
