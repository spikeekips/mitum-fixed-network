package basicstates

import (
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
)

var EmptySwitchFunc = func() error { return nil }

type State interface {
	Enter(StateSwitchContext) (func() error, error)
	Exit(StateSwitchContext) (func() error, error)
	ProcessProposal(ballot.Proposal) error
	ProcessVoteproof(base.Voteproof) error
	SetStates(*States) State
}

type BaseState struct {
	state                 base.State
	States                *States
	lastVoteproofFunc     func() base.Voteproof
	lastINITVoteproofFunc func() base.Voteproof
	setLastVoteproofFunc  func(base.Voteproof) bool
	newProposalFunc       func(ballot.Proposal)
	newVoteproofFunc      func(base.Voteproof)
	broadcastSealsFunc    func(seal.Seal, bool /* to local */) error
	timers                *localtime.Timers
	switchStateFunc       func(StateSwitchContext) error
	newBlocksFunc         func([]block.Block) error
	enterFunc             func(StateSwitchContext) (func() error, error)
	exitFunc              func(StateSwitchContext) (func() error, error)
	processVoteproofFunc  func(base.Voteproof) error
	syncableChannelsFunc  func() map[string]network.Channel
	exiting               *util.LockedItem
}

func NewBaseState(st base.State) *BaseState {
	return &BaseState{
		state:   st,
		exiting: util.NewLockedItem(false),
	}
}

func (st *BaseState) Enter(sctx StateSwitchContext) (func() error, error) {
	if sctx.ToState() != st.state {
		return nil, errors.Errorf("context not for entering this state, %v", st.state)
	}

	_ = st.exiting.Set(false)

	if st.enterFunc != nil {
		return st.enterFunc(sctx)
	}

	return nil, nil
}

func (st *BaseState) Exit(sctx StateSwitchContext) (func() error, error) {
	if sctx.FromState() != st.state {
		return nil, errors.Errorf("context not for exiting this state, %v", st.state)
	}

	if st.exitFunc != nil {
		return st.exitFunc(sctx)
	}

	return nil, nil
}

func (*BaseState) ProcessProposal(ballot.Proposal) error { return nil }
func (st *BaseState) ProcessVoteproof(voteproof base.Voteproof) error {
	if st.processVoteproofFunc != nil {
		return st.processVoteproofFunc(voteproof)
	}

	return nil
}

func (st *BaseState) SetStates(ss *States) State {
	st.States = ss

	return st
}

func (st *BaseState) LastVoteproof() base.Voteproof {
	if st.lastVoteproofFunc != nil {
		return st.lastVoteproofFunc()
	}

	return st.States.LastVoteproof()
}

func (st *BaseState) LastINITVoteproof() base.Voteproof {
	if st.lastINITVoteproofFunc != nil {
		return st.lastINITVoteproofFunc()
	}

	return st.States.LastINITVoteproof()
}

func (st *BaseState) SetLastVoteproof(voteproof base.Voteproof) bool {
	if st.setLastVoteproofFunc != nil {
		return st.setLastVoteproofFunc(voteproof)
	}

	return st.States.SetLastVoteproof(voteproof)
}

func (st *BaseState) NewProposal(proposal ballot.Proposal) {
	if st.newVoteproofFunc != nil {
		st.newProposalFunc(proposal)

		return
	}

	st.States.NewProposal(proposal)
}

func (st *BaseState) NewVoteproof(voteproof base.Voteproof) {
	if st.newVoteproofFunc != nil {
		st.newVoteproofFunc(voteproof)

		return
	}

	st.States.NewVoteproof(voteproof)
}

func (st *BaseState) BroadcastBallot(blt ballot.Ballot, toLocal bool) error {
	if st.broadcastSealsFunc != nil {
		return st.broadcastSealsFunc(blt, toLocal)
	}

	st.States.BroadcastBallot(blt, toLocal)

	return nil
}

func (st *BaseState) Timers() *localtime.Timers {
	if st.timers != nil {
		return st.timers
	}

	return st.States.Timers()
}

func (st *BaseState) StateSwitch(sctx StateSwitchContext) error {
	if st.switchStateFunc != nil {
		return st.switchStateFunc(sctx)
	}

	return st.States.SwitchState(sctx)
}

func (st *BaseState) NewBlocks(blks []block.Block) error {
	if st.newBlocksFunc != nil {
		return st.newBlocksFunc(blks)
	}

	return st.States.NewBlocks(blks)
}

func (st *BaseState) NewStateSwitchContext(next base.State) StateSwitchContext {
	return NewStateSwitchContext(st.state, next)
}

func (st *BaseState) syncableChannels() map[string]network.Channel {
	if st.syncableChannelsFunc != nil {
		return st.syncableChannelsFunc()
	}

	if st.States == nil {
		return nil
	}

	pn := map[string]network.Channel{}
	st.States.nodepool.TraverseAliveRemotes(func(no base.Node, ch network.Channel) bool {
		pn[no.String()] = ch

		return true
	})

	if ch := st.oldNode(); ch != nil {
		pn["handover"] = ch
	}

	return pn
}

func (st *BaseState) oldNode() network.Channel {
	if !st.underHandover() {
		return nil
	}

	return st.States.hd.OldNode()
}

func (st *BaseState) underHandover() bool {
	return st.States != nil && st.States.underHandover()
}

var ConsensusStuckError = util.NewError("consensus looks stuck")

type ConsensusStuckChecker struct {
	lastVoteproof base.Voteproof
	state         base.State
	endure        time.Duration
	nowFunc       func() time.Time
}

func NewConsensusStuckChecker(
	lastVoteproof base.Voteproof,
	state base.State,
	endure time.Duration,
) *ConsensusStuckChecker {
	return &ConsensusStuckChecker{
		lastVoteproof: lastVoteproof,
		state:         state,
		endure:        endure,
	}
}

func (cc *ConsensusStuckChecker) IsValidState() (bool, error) {
	if cc.state != base.StateConsensus {
		return false, nil
	}

	return true, nil
}

func (cc *ConsensusStuckChecker) IsOldLastVoteproofTime() (bool, error) {
	if cc.lastVoteproof == nil {
		return true, nil
	}

	var now time.Time
	if cc.nowFunc != nil {
		now = cc.nowFunc()
	} else {
		now = localtime.UTCNow()
	}

	if since := now.Sub(cc.lastVoteproof.FinishedAt()); since > cc.endure {
		return false, ConsensusStuckError.Errorf(
			"last voteproof is too old, %s from now", since.String())
	}

	return true, nil
}
