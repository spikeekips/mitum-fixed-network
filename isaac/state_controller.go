package isaac

import (
	"fmt"
	"sync"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/node"
	"github.com/spikeekips/mitum/seal"
)

type StateController struct {
	sync.RWMutex
	*common.Logger
	homeState        *HomeState
	compiler         *Compiler
	sealStorage      SealStorage
	chanState        chan StateContext
	bootingHandler   StateHandler
	joinHandler      StateHandler
	consensusHandler StateHandler
	stoppedHandler   StateHandler
	stateHandler     StateHandler
}

func NewStateController(
	homeState *HomeState,
	compiler *Compiler,
	sealStorage SealStorage,
	bootingHandler StateHandler,
	joinHandler StateHandler,
	consensusHandler StateHandler,
	stoppedHandler StateHandler,
) *StateController {
	chanState := make(chan StateContext)
	sc := &StateController{
		Logger:           common.NewLogger(log, "module", "state-controller"),
		homeState:        homeState,
		compiler:         compiler,
		sealStorage:      sealStorage,
		chanState:        chanState,
		bootingHandler:   bootingHandler.SetChanState(chanState),
		joinHandler:      joinHandler.SetChanState(chanState),
		consensusHandler: consensusHandler.SetChanState(chanState),
		stoppedHandler:   stoppedHandler.SetChanState(chanState),
	}

	return sc
}

func (sc *StateController) Start() error {
	go sc.loopState()

	// start booting
	if err := sc.setState(NewStateContext(node.StateBooting)); err != nil {
		return err
	}

	return nil
}

func (sc *StateController) Stop() error {
	sc.Lock()
	defer sc.Unlock()

	close(sc.chanState)

	if err := sc.stateHandler.Deactivate(); err != nil {
		return err
	}

	sc.stateHandler = nil

	return nil
}

func (sc *StateController) loopState() {
	for sct := range sc.chanState {
		current := sc.homeState.State()
		if err := sc.setState(sct); err != nil {
			sc.Log().Error(
				"error change state",
				"error", err,
				"current_state", current,
				"new_state", sct.State(),
			)
		} else {
			sc.Log().Debug(
				"state changed",
				"current_state", current,
				"new_state", sct.State(),
			)
		}
	}
}

func (sc *StateController) setState(sct StateContext) error {
	if err := sct.State().IsValid(); err != nil {
		return err
	}

	sc.Lock()
	defer sc.Unlock()

	if sc.stateHandler != nil && sc.stateHandler.State() == sct.State() {
		return xerrors.Errorf("same state")
	}

	// stop previous StateHandler and start new StateHandler
	if sc.stateHandler != nil {
		if err := sc.stateHandler.Deactivate(); err != nil {
			return err
		}
	}

	var handler StateHandler
	switch sct.State() {
	case node.StateBooting:
		handler = sc.bootingHandler
	case node.StateJoin:
		handler = sc.joinHandler
	case node.StateConsensus:
		handler = sc.consensusHandler
	case node.StateStopped:
		handler = sc.stoppedHandler
	default:
		return xerrors.Errorf("handler not found for state; state=%v", sct.State())
	}

	if err := handler.Activate(sct); err != nil {
		return err
	}

	_ = sc.homeState.SetState(sct.State())
	sc.stateHandler = handler

	return nil
}

func (sc *StateController) Receive(message interface{}) error {
	sl, ok := message.(seal.Seal)
	if !ok {
		sc.Log().Error("receive unknown message", "message", message)
		return xerrors.Errorf("receive unknown message; message=%q", message)
	}

	sc.Log().Debug(
		fmt.Sprintf("seal received; %v", sl.Type()),
		"seal", sl,
	)

	if err := sl.IsValid(); err != nil {
		sc.Log().Error("invalid seal", "seal", sl.Hash(), "error", err)
		return err
	}

	// save seal
	if err := sc.sealStorage.Save(sl); err != nil {
		return err
	}

	switch sl.Type() {
	case ProposalType:
		proposal, ok := sl.(Proposal)
		if !ok {
			return xerrors.Errorf("seal.Type() is proposal, but it's not; message=%q", message)
		}

		if err := sc.handleProposal(proposal); err != nil {
			return err
		}
	case BallotType:
		ballot, ok := sl.(Ballot)
		if !ok {
			return xerrors.Errorf("seal.Type() is ballot, but it's not; message=%q", message)
		}

		if err := sc.handleBallot(ballot); err != nil {
			return err
		}
	}

	return nil
}

func (sc *StateController) handleProposal(proposal Proposal) error {
	// TODO check proposal

	// hand over VoteResult to StateHandler
	sc.RLock()
	handler := sc.stateHandler
	sc.RUnlock()

	if handler != nil {
		if err := handler.ReceiveProposal(proposal); err != nil {
			return err
		}
	}

	return nil
}

func (sc *StateController) StateHandler() StateHandler {
	sc.RLock()
	defer sc.RUnlock()
	return sc.stateHandler
}

func (sc *StateController) handleBallot(ballot Ballot) error {
	vr, err := sc.compiler.Vote(ballot)
	if err != nil {
		sc.Log().Error("failed to vote ballot", "ballot", ballot.Hash(), "error", err)
		return err
	}

	if !vr.IsClosed() && vr.IsFinished() {
		// hand over VoteResult to StateHandler
		if err := sc.StateHandler().ReceiveVoteResult(vr); err != nil {
			return err
		}
	}

	return nil
}
