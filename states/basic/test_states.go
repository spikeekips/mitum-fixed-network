//go:build test
// +build test

package basicstates

import (
	"sync"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/network/discovery"
	"github.com/spikeekips/mitum/util/localtime"
)

func (st *BaseState) SetLastVoteproofFuncs(last func() base.Voteproof, lastinit func() base.Voteproof, setter func(base.Voteproof) bool) {
	st.lastVoteproofFunc = last
	st.lastINITVoteproofFunc = lastinit
	st.setLastVoteproofFunc = setter
}

func (st *BaseState) SetNewProposalFunc(fn func(ballot.Proposal)) {
	st.newProposalFunc = fn
}

func (st *BaseState) SetNewVoteproofFunc(fn func(base.Voteproof)) {
	st.newVoteproofFunc = fn
}

func (st *BaseState) SetBroadcastSealsFunc(fn func(seal.Seal, bool) error) {
	st.broadcastSealsFunc = fn
}

func (st *BaseState) SetTimers(timers *localtime.Timers) {
	st.timers = timers
}

func (st *BaseState) SetStateSwitchFunc(fn func(StateSwitchContext) error) {
	st.switchStateFunc = fn
}

func (st *BaseState) SetNewBlocksFunc(fn func([]block.Block) error) {
	st.newBlocksFunc = fn
}

func (st *BaseState) SetProcessVoteproofFunc(fn func(base.Voteproof) error) {
	st.processVoteproofFunc = fn
}

func (st *BaseState) SetEnterFunc(fn func(StateSwitchContext) (func() error, error)) {
	st.enterFunc = fn
}

func (st *BaseState) SetExitFunc(fn func(StateSwitchContext) (func() error, error)) {
	st.exitFunc = fn
}

type baseTestState struct {
	sync.Mutex
	isaac.BaseTest
	local  *isaac.Local
	remote *isaac.Local
}

func (t *baseTestState) SetupTest() {
	t.BaseTest.SetupTest()

	ls := t.Locals(2)
	t.local, t.remote = ls[0], ls[1]
}

func (t *baseTestState) exitState(state State, sctx StateSwitchContext) {
	f, err := state.Exit(sctx)
	if err != nil {
		t.T().Log("error:", err)
	}

	if f != nil {
		if err := f(); err != nil {
			t.T().Log("error:", err)
		}
	}
}

func (t *baseTestState) newStates(local *isaac.Local, suffrage base.Suffrage, st State) *States {
	if st == nil {
		st = NewBaseState(base.StateHandover)
	}

	stt, err := NewStates(local.Database(), local.Policy(), local.Nodes(), suffrage, nil,
		NewBaseState(base.StateStopped),
		NewBaseState(base.StateBooting),
		NewBaseState(base.StateJoining),
		NewBaseState(base.StateConsensus),
		NewBaseState(base.StateSyncing),
		st,
		nil, nil,
	)
	t.NoError(err)

	return stt
}

func (t *baseTestState) newChannel(s string) network.Channel {
	ci, err := network.NewHTTPConnInfoFromString(s, true)
	t.NoError(err)

	ch, err := discovery.LoadNodeChannel(ci, t.Encs, time.Second*2)
	t.NoError(err)

	return ch
}
