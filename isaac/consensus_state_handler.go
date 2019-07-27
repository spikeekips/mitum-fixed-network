package isaac

import (
	"sync"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/node"
	"golang.org/x/xerrors"
)

type ConsensusStateHandler struct {
	sync.RWMutex
	*common.Logger
	homeState         *HomeState
	compiler          *Compiler
	nt                network.Network
	started           bool
	chanState         chan node.State
	timer             *common.CallbackTimer
	voteResultChecker *common.ChainChecker
}

func NewConsensusStateHandler(
	homeState *HomeState,
	compiler *Compiler,
	nt network.Network,
) (*ConsensusStateHandler, error) {
	if homeState.PreviousBlock().Empty() {
		return nil, xerrors.Errorf("previous block is empty")
	}

	logger := common.NewLogger(log, "module", "consensus-state-handler")

	cs := &ConsensusStateHandler{
		Logger:            logger,
		homeState:         homeState,
		compiler:          compiler,
		nt:                nt,
		voteResultChecker: NewConsensusVoteResultChecker(homeState),
	}

	return cs, nil
}

func (cs *ConsensusStateHandler) Start() error {
	_ = cs.Stop()

	cs.Lock()
	defer cs.Unlock()
	cs.started = true

	return nil
}

func (cs *ConsensusStateHandler) Stop() error {
	if err := cs.Deactivate(); err != nil {
		return err
	}

	cs.Lock()
	defer cs.Unlock()
	cs.started = false

	return nil
}

func (cs *ConsensusStateHandler) IsStopped() bool {
	cs.RLock()
	defer cs.RUnlock()

	return !cs.started
}

func (cs *ConsensusStateHandler) Activate() error {
	_ = cs.stopTimer()

	return nil
}

func (cs *ConsensusStateHandler) Deactivate() error {
	return cs.stopTimer()
}

func (cs *ConsensusStateHandler) SetChanState(ch chan node.State) StateHandler {
	cs.chanState = ch
	return cs
}

func (cs *ConsensusStateHandler) State() node.State {
	return node.StateConsensus
}

func (cs *ConsensusStateHandler) stopTimer() error {
	cs.RLock()
	defer cs.RUnlock()

	if cs.timer == nil || cs.timer.IsStopped() {
		return nil
	}

	if err := cs.timer.Stop(); err != nil {
		cs.Log().Error("failed to stop timer", "error", err)
		return err
	}

	return nil
}

func (cs *ConsensusStateHandler) ReceiveProposal(proposal Proposal) error {
	return nil
}

func (cs *ConsensusStateHandler) ReceiveVoteResult(vr VoteResult) error {
	err := cs.voteResultChecker.
		New(nil).
		SetContext("vr", vr).
		SetContext("lastINITVoteResult", cs.compiler.LastINITVoteResult()).
		Check()
	if err != nil {
		return err
	}

	if !vr.GotMajority() {
		cs.Log().Debug("got not majority; ignore", "vr", vr)
		return nil
	}

	if vr.Stage() == StageINIT {
		return cs.gotMajorityINIT(vr)
	} else {
		return cs.gotMajorityStages(vr)
	}

	return nil
}

func (cs *ConsensusStateHandler) gotMajorityINIT(vr VoteResult) error {
	_ = cs.stopTimer()

	diff := vr.Height().Sub(cs.homeState.Block().Height()).Int64()
	switch {
	case diff == 2: // expected; move to consensus
		cs.Log().Debug("got VoteResult of next block; keep going", "vr", vr)
	case diff == 1: // expected; move to consensus
		cs.Log().Debug("got VoteResult of current block; keep going", "vr", vr)
		go cs.startNewRound(vr)
	default: // higher height received, move to sync
		cs.Log().Debug("got not expected height VoteResult; move to sync", "vr", vr)
		cs.chanState <- node.StateSync
		return nil
	}

	// TODO store new block

	return nil
}

func (cs *ConsensusStateHandler) startNewRound(vr VoteResult) error {
	// TODO

	return nil
}

func (cs *ConsensusStateHandler) gotMajorityStages(vr VoteResult) error {
	// TODO

	return nil
}
