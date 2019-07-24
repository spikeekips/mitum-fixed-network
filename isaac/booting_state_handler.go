package isaac

import (
	"sync"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/node"
)

type BootingStateHandler struct {
	sync.RWMutex
	*common.Logger
	started         bool
	chanState       chan node.State
	proposalChecker *common.ChainChecker
}

func NewBootingStateHandler(homeState *HomeState) *BootingStateHandler {
	return &BootingStateHandler{
		Logger:          common.NewLogger(log, "module", "booting-state-handler"),
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

func (bs *BootingStateHandler) Activate() error {
	go func() {
		bs.chanState <- node.StateJoin
	}()

	return nil
}

func (bs *BootingStateHandler) Deactivate() error {
	return nil
}

func (bs *BootingStateHandler) SetChanState(ch chan node.State) StateHandler {
	bs.chanState = ch
	return bs
}

func (bs *BootingStateHandler) State() node.State {
	return node.StateBooting
}

func (bs *BootingStateHandler) ReceiveProposal(proposal Proposal) error {
	err := bs.proposalChecker.
		New(nil).
		SetContext(
			"proposal", proposal,
		).
		Check()

	if err != nil {
		return err
	}

	return nil
}

func (bs *BootingStateHandler) ReceiveVoteResult(vr VoteResult) error {
	return nil
}
