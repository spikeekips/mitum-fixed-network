package isaac

import (
	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/node"
)

type BootingStateHandler struct {
	*common.Logger
	*common.ReaderDaemon
	chanState       chan node.State
	proposalChecker *common.ChainChecker
}

func NewBootingStateHandler(homeState *HomeState) *BootingStateHandler {
	bs := &BootingStateHandler{
		Logger:          common.NewLogger(log, "module", "booting-state-handler"),
		proposalChecker: NewProposalCheckerBooting(homeState),
	}
	bs.ReaderDaemon = common.NewReaderDaemon(true, 0, nil)

	return bs
}

func (bs *BootingStateHandler) Start() error {
	if err := bs.ReaderDaemon.Start(); err != nil {
		return err
	}

	// TODO health check:
	//	- blocks in storage is valid
	//	- network status
	//	- connectivity of suffrage members
	//	- etc

	// NOTE everything ok, move to next state, JOIN
	go func() {
		bs.chanState <- node.StateJoin
	}()

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
