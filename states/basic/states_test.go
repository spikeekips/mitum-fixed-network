package basicstates

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	channetwork "github.com/spikeekips/mitum/network/gochan"
	"github.com/spikeekips/mitum/states"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testStates struct {
	baseTestState
}

func (t *testStates) newStates() *States {
	suffrage := t.Suffrage(t.remote, t.local)
	ss, err := NewStates(
		t.local.Database(),
		t.local.Policy(),
		t.local.Nodes(),
		suffrage,
		t.Ballotbox(suffrage, t.local.Policy()),
		NewBaseState(base.StateStopped),
		NewBaseState(base.StateBooting),
		NewBaseState(base.StateJoining),
		NewBaseState(base.StateConsensus),
		NewBaseState(base.StateSyncing),
		NewBaseState(base.StateHandover),
		nil,
		nil,
	)
	t.NoError(err)
	t.NotNil(ss)

	return ss
}

func (t *testStates) TestLastINITVoteproof() {
	suffrage := t.Suffrage(t.remote, t.local)
	ss, err := NewStates(
		t.local.Database(),
		t.local.Policy(),
		t.local.Nodes(),
		suffrage,
		t.Ballotbox(suffrage, t.local.Policy()),
		NewBaseState(base.StateStopped),
		NewBaseState(base.StateBooting),
		NewBaseState(base.StateJoining),
		NewBaseState(base.StateConsensus),
		NewBaseState(base.StateSyncing),
		NewBaseState(base.StateHandover),
		nil,
		nil,
	)
	t.NoError(err)
	t.NotNil(ss)

	livp := t.local.Database().LastVoteproof(base.StageINIT)
	t.NotNil(livp)

	sslivp := ss.LastINITVoteproof()
	t.Equal(livp.Bytes(), sslivp.Bytes())

	t.Equal(base.StateStopped, ss.State())
}

func (t *testStates) TestNewSealVoteproof() {
	ss := t.newStates()
	defer func() {
		_ = ss.Stop()
	}()

	gotvoteproofch := make(chan base.Voteproof)
	stateStopped := NewBaseState(base.StateStopped)
	stateStopped.SetProcessVoteproofFunc(func(voteproof base.Voteproof) error {
		gotvoteproofch <- voteproof

		return nil
	})

	ss.states[base.StateStopped] = stateStopped

	stopch := make(chan error)
	go func() {
		stopch <- ss.Start()
	}()

	ibl := t.NewINITBallot(t.local, base.Round(0), nil)
	ibr := t.NewINITBallot(t.remote, ibl.Fact().Round(), nil)

	t.NoError(ss.NewSeal(ibl))
	t.NoError(ss.NewSeal(ibr))

	select {
	case err := <-stopch:
		t.NoError(fmt.Errorf("stopped: %w", err))
	case voteproof := <-gotvoteproofch:
		lib := ss.ballotbox.LatestBallot()
		t.NotNil(lib)

		t.NotNil(voteproof)

		t.Equal(base.StageINIT, voteproof.Stage())
		t.Equal(ibl.Fact().Height(), voteproof.Height())
		t.Equal(ibl.Fact().Round(), voteproof.Round())
		t.Equal(base.VoteResultMajority, voteproof.Result())
		t.NotNil(voteproof.Majority())
	}
}

func (t *testStates) TestSwitchingState() {
	ss := t.newStates()
	defer func() {
		_ = ss.Stop()
	}()

	statech := make(chan StateSwitchContext)
	stateConsensus := NewBaseState(base.StateConsensus)
	stateConsensus.SetEnterFunc(func(sctx StateSwitchContext) (func() error, error) {
		statech <- sctx
		return nil, nil
	})

	ss.states[base.StateConsensus] = stateConsensus

	stopch := make(chan error)
	go func() {
		stopch <- ss.Start()
	}()

	sctx := NewStateSwitchContext(ss.State(), base.StateConsensus)
	t.NoError(ss.SwitchState(sctx))

	select {
	case err := <-stopch:
		t.NoError(fmt.Errorf("stopped: %w", err))
	case nsctx := <-statech:
		t.Equal(sctx.FromState(), nsctx.FromState())
		t.Equal(sctx.ToState(), nsctx.ToState())
	}
}

func (t *testStates) TestSwitchingUnknownState() {
	ss := t.newStates()
	defer func() {
		_ = ss.Stop()
	}()

	statech := make(chan StateSwitchContext)
	stateConsensus := NewBaseState(base.StateConsensus)
	stateConsensus.SetEnterFunc(func(sctx StateSwitchContext) (func() error, error) {
		statech <- sctx
		return nil, nil
	})

	ss.states[base.StateConsensus] = stateConsensus

	stopch := make(chan error)
	go func() {
		stopch <- ss.Start()
	}()

	prev := ss.State()
	sctx := NewStateSwitchContext(base.StateEmpty, base.StateConsensus)
	err := ss.SwitchState(sctx)
	t.NotNil(err)
	t.Contains(err.Error(), "invalid state found")

	sctx = NewStateSwitchContext(base.StateEmpty, base.StateConsensus).allowEmpty(true)
	t.NoError(ss.SwitchState(sctx))

	select {
	case err := <-stopch:
		t.NoError(fmt.Errorf("stopped: %w", err))
	case nsctx := <-statech:
		t.Equal(prev, nsctx.FromState())
		t.Equal(sctx.ToState(), nsctx.ToState())
	}
}

func (t *testStates) TestSwitchingStateWithVoteproof() {
	ss := t.newStates()
	defer func() {
		_ = ss.Stop()
	}()

	statech := make(chan StateSwitchContext)
	stateConsensus := NewBaseState(base.StateConsensus)
	stateConsensus.SetEnterFunc(func(sctx StateSwitchContext) (func() error, error) {
		statech <- sctx
		return nil, nil
	})

	gotvoteproofch := make(chan base.Voteproof)
	stateConsensus.SetProcessVoteproofFunc(func(voteproof base.Voteproof) error {
		gotvoteproofch <- voteproof

		return nil
	})

	ss.states[base.StateConsensus] = stateConsensus

	stopch := make(chan error)
	go func() {
		stopch <- ss.Start()
	}()

	sctx := NewStateSwitchContext(ss.State(), base.StateConsensus)
	t.NoError(ss.SwitchState(sctx))

	select {
	case err := <-stopch:
		t.NoError(fmt.Errorf("stopped: %w", err))
	case nsctx := <-statech:
		t.Equal(sctx.FromState(), nsctx.FromState())
		t.Equal(sctx.ToState(), nsctx.ToState())
	}

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	ss.NewVoteproof(ivp)

	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("timeout to wait voteproof thru consensus state"))
	case voteproof := <-gotvoteproofch:
		t.NotNil(voteproof)

		t.Equal(base.StageINIT, voteproof.Stage())
		t.Equal(ivp.Height(), voteproof.Height())
		t.Equal(ivp.Round(), voteproof.Round())
		t.Equal(base.VoteResultMajority, voteproof.Result())
		t.NotNil(voteproof.Majority())
	}
}

func (t *testStates) TestNewVoteproofThruStateSwithContext() {
	ss := t.newStates()
	defer func() {
		_ = ss.Stop()
	}()

	statech := make(chan StateSwitchContext)
	stateConsensus := NewBaseState(base.StateConsensus)
	stateConsensus.SetEnterFunc(func(sctx StateSwitchContext) (func() error, error) {
		statech <- sctx
		return nil, nil
	})

	gotvoteproofch := make(chan base.Voteproof)
	stateConsensus.SetProcessVoteproofFunc(func(voteproof base.Voteproof) error {
		gotvoteproofch <- voteproof

		return nil
	})

	ss.states[base.StateConsensus] = stateConsensus

	stopch := make(chan error)
	go func() {
		stopch <- ss.Start()
	}()

	sctx := NewStateSwitchContext(ss.State(), base.StateConsensus)
	t.NoError(ss.SwitchState(sctx))

	select {
	case err := <-stopch:
		t.NoError(fmt.Errorf("stopped: %w", err))
	case nsctx := <-statech:
		t.Equal(sctx.FromState(), nsctx.FromState())
		t.Equal(sctx.ToState(), nsctx.ToState())
	}

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	t.NoError(ss.SwitchState(NewStateSwitchContext(base.StateConsensus, base.StateConsensus).SetVoteproof(ivp)))

	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("timeout to wait voteproof thru consensus state"))
	case voteproof := <-gotvoteproofch:
		t.NotNil(voteproof)

		t.Equal(base.StageINIT, voteproof.Stage())
		t.Equal(ivp.Height(), voteproof.Height())
		t.Equal(ivp.Round(), voteproof.Round())
		t.Equal(base.VoteResultMajority, voteproof.Result())
		t.NotNil(voteproof.Majority())
	}
}

func (t *testStates) TestNewVoteproofFromBallot() {
	ss := t.newStates()
	defer func() {
		_ = ss.Stop()
	}()

	statech := make(chan StateSwitchContext)
	stateConsensus := NewBaseState(base.StateConsensus)
	stateConsensus.SetEnterFunc(func(sctx StateSwitchContext) (func() error, error) {
		statech <- sctx
		return nil, nil
	})

	gotvoteproofch := make(chan base.Voteproof)
	stateConsensus.SetProcessVoteproofFunc(func(voteproof base.Voteproof) error {
		gotvoteproofch <- voteproof

		return nil
	})

	ss.states[base.StateConsensus] = stateConsensus

	stopch := make(chan error)
	go func() {
		stopch <- ss.Start()
	}()

	sctx := NewStateSwitchContext(ss.State(), base.StateConsensus)
	t.NoError(ss.SwitchState(sctx))

	select {
	case err := <-stopch:
		t.NoError(fmt.Errorf("stopped: %w", err))
	case nsctx := <-statech:
		t.Equal(sctx.FromState(), nsctx.FromState())
		t.Equal(sctx.ToState(), nsctx.ToState())
	}

	normalib := t.NewINITBallot(t.local, base.Round(0), nil)
	t.NoError(ss.NewSeal(normalib)) // NOTE this will be voted, but it's voteproof will be ignored

	ivp, err := t.NewVoteproof(base.StageINIT, normalib.Fact(), t.local, t.remote)
	t.NoError(err)

	// NOTE ballot from other nodes can be handled
	ab := t.NewACCEPTBallot(t.remote, base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256(), ivp)
	t.NoError(ss.NewSeal(ab))

	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("timeout to wait voteproof thru consensus state"))
	case voteproof := <-gotvoteproofch:
		t.NotNil(voteproof)

		t.Equal(base.StageINIT, voteproof.Stage())
		t.Equal(ivp.Height(), voteproof.Height())
		t.Equal(ivp.Round(), voteproof.Round())
		t.Equal(base.VoteResultMajority, voteproof.Result())
		t.NotNil(voteproof.Majority())
	}
}

// TestFailedSwitchingState tests,
// - failed to switch state
// - states will moves state to booting
func (t *testStates) TestFailedSwitchingState() {
	ss := t.newStates()
	defer func() {
		_ = ss.Stop()
	}()

	statech := make(chan StateSwitchContext)

	stateBooting := NewBaseState(base.StateBooting)
	stateBooting.SetEnterFunc(func(sctx StateSwitchContext) (func() error, error) {
		statech <- sctx

		return nil, nil
	})

	stateJoining := NewBaseState(base.StateJoining)
	stateJoining.SetEnterFunc(func(StateSwitchContext) (func() error, error) {
		return nil, errors.Errorf("born to be killed")
	})

	ss.states[base.StateBooting] = stateBooting
	ss.states[base.StateJoining] = stateJoining

	stopch := make(chan error)
	go func() {
		stopch <- ss.Start()
	}()

	sctx := NewStateSwitchContext(ss.State(), base.StateJoining)
	t.NoError(ss.SwitchState(sctx))

	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("timeout to wait to switch state"))
	case err := <-stopch:
		t.NoError(fmt.Errorf("stopped: %w", err))
	case sctx := <-statech:
		t.Equal(base.StateBooting, sctx.ToState())
	}
}

// TestFailedSwitchingState tests,
// - failed to switch state
// - states will moves state to booting
// - if switching to booting also failed, states will be stopped with error
func (t *testStates) TestFailedSwitchingStateButKeepFailing() {
	ss := t.newStates()
	defer func() {
		_ = ss.Stop()
	}()

	var trying int64
	stateBooting := NewBaseState(base.StateBooting)
	stateBooting.SetEnterFunc(func(sctx StateSwitchContext) (func() error, error) {
		atomic.AddInt64(&trying, 1)

		return nil, errors.Errorf("impossible entering")
	})

	stateJoining := NewBaseState(base.StateJoining)
	stateJoining.SetEnterFunc(func(StateSwitchContext) (func() error, error) {
		return nil, errors.Errorf("born to be killed")
	})

	ss.states[base.StateBooting] = stateBooting
	ss.states[base.StateJoining] = stateJoining

	stopch := make(chan error)
	go func() {
		stopch <- ss.Start()
	}()

	sctx := NewStateSwitchContext(ss.State(), base.StateJoining)
	t.NoError(ss.SwitchState(sctx))

	select {
	case <-time.After(time.Second * 4):
		t.NoError(errors.Errorf("timeout to wait states to be stopped"))
	case err := <-stopch:
		t.Contains(err.Error(), "failed to move to booting")
		t.Contains(err.Error(), "impossible entering")
	}

	t.Equal(int64(1), atomic.LoadInt64(&trying))
}

func (t *testStates) TestFailedSwitchingStateButIgnore() {
	ss := t.newStates()

	stateStopped := NewBaseState(base.StateStopped)
	stateStopped.SetExitFunc(func(sctx StateSwitchContext) (func() error, error) {
		return func() error {
			return errors.Errorf("exit error")
		}, nil
	})

	stateJoining := NewBaseState(base.StateJoining)
	stateJoining.SetEnterFunc(func(StateSwitchContext) (func() error, error) {
		return func() error {
			return errors.Errorf("enter error")
		}, nil
	})

	ss.states[base.StateStopped] = stateStopped
	ss.states[base.StateJoining] = stateJoining

	go func() {
		_ = ss.Start()
	}()
	defer func() {
		_ = ss.Stop()
	}()

	sctx := NewStateSwitchContext(ss.State(), base.StateJoining)
	t.NoError(ss.SwitchState(sctx))

	t.Equal(base.StateJoining, sctx.ToState())
}

func (t *testStates) TestFailedSwitchingStateButSameState() {
	ss := t.newStates()

	statech := make(chan StateSwitchContext)

	stateBooting := NewBaseState(base.StateBooting)
	stateBooting.SetEnterFunc(func(sctx StateSwitchContext) (func() error, error) {
		statech <- sctx

		return nil, nil
	})

	stateJoining := NewBaseState(base.StateJoining)
	stateJoining.SetEnterFunc(func(StateSwitchContext) (func() error, error) {
		return func() error {
			return NewStateSwitchContext(ss.State(), base.StateJoining)
		}, nil
	})

	ss.states[base.StateBooting] = stateBooting
	ss.states[base.StateJoining] = stateJoining

	go func() {
		_ = ss.Start()
	}()
	defer func() {
		_ = ss.Stop()
	}()

	sctx := NewStateSwitchContext(ss.State(), base.StateJoining)
	t.NoError(ss.SwitchState(sctx))

	t.Equal(base.StateJoining, sctx.ToState())
}

func (t *testStates) TestNewBallotNoneSuffrage() {
	suffrage := t.Suffrage(t.remote)
	ss, err := NewStates(
		t.local.Database(),
		t.local.Policy(),
		t.local.Nodes(),
		suffrage,
		t.Ballotbox(suffrage, t.local.Policy()),
		NewBaseState(base.StateStopped),
		NewBaseState(base.StateBooting),
		NewBaseState(base.StateJoining),
		NewBaseState(base.StateConsensus),
		NewBaseState(base.StateSyncing),
		NewBaseState(base.StateHandover),
		nil,
		nil,
	)
	t.NoError(err)
	t.NotNil(ss)

	defer func() {
		_ = ss.Stop()
	}()

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	t.NoError(ss.NewSeal(ib))

	<-time.After(time.Second * 1)

	// NOTE in none-suffrage node, new incoming ballot will not be voted
	lib := ss.ballotbox.LatestBallot()
	t.Nil(lib)
}

func (t *testStates) TestNewOperationSealNoneSuffrage() {
	remotech := channetwork.NewChannel(0, t.remote.Channel().ConnInfo())
	_ = t.remote.SetChannel(remotech)
	t.local.Nodes().SetChannel(t.remote.Node().Address(), remotech)

	// NOTE local State
	ss, err := NewStates(
		t.local.Database(),
		t.local.Policy(),
		t.local.Nodes(),
		t.Suffrage(t.remote), // NOTE local is not in suffrage
		nil,
		NewBaseState(base.StateStopped),
		NewBaseState(base.StateBooting),
		NewBaseState(base.StateJoining),
		NewBaseState(base.StateConsensus),
		NewBaseState(base.StateSyncing),
		NewBaseState(base.StateHandover),
		nil,
		nil,
	)
	t.NoError(err)
	t.NotNil(ss)

	defer func() {
		_ = ss.Stop()
	}()

	op, err := operation.NewKVOperation(t.local.Node().Privatekey(), util.UUID().Bytes(), util.UUID().String(), []byte(util.UUID().String()), nil)
	t.NoError(err)

	sl, err := operation.NewBaseSeal(t.local.Node().Privatekey(), []operation.Operation{op}, t.local.Policy().NetworkID())
	t.NoError(err)

	t.NoError(ss.NewSeal(sl))

	select {
	case <-time.After(time.Second * 10):
		t.NoError(errors.Errorf("waited broadcasted seal, but nothing"))
	case rsl := <-remotech.ReceiveSeal():
		t.True(sl.Hash().Equal(rsl.Hash()))
	}
}

func (t *testStates) TestSwitchingStateToHandoverNotHandoverReady() {
	ss := t.newStates()
	defer func() {
		_ = ss.Stop()
	}()

	stateHandover := NewBaseState(base.StateHandover)

	ss.states[base.StateHandover] = stateHandover

	ss.hd = NewHandover(nil, t.Encs, t.local.Policy(), t.local.Nodes(), ss.suffrage)

	old := t.newChannel("https://old")
	ss.hd.checkDuplicatedNodeFunc = func() (network.Channel, network.NodeInfo, error) { return old, network.NodeInfoV0{}, nil }

	stopch := make(chan error)
	go func() {
		stopch <- ss.Start()
	}()

	{
		ticker := time.NewTicker(time.Millisecond * 300)
		for range ticker.C {
			if ss.underHandover() {
				break
			}
		}

		ticker.Stop()
		t.T().Log("under handover")
	}

	sctx := NewStateSwitchContext(ss.State(), base.StateHandover)

	t.NoError(ss.switchState(sctx))
	t.Equal(base.StateSyncing, ss.State())
}

func (t *testStates) TestSwitchingStateToHandoverUnderHandoverNotHandoverReady() {
	ss := t.newStates()
	defer func() {
		_ = ss.Stop()
	}()

	stateConsensus := NewBaseState(base.StateConsensus)
	stateHandover := NewBaseState(base.StateHandover)

	ss.states[base.StateConsensus] = stateConsensus
	ss.states[base.StateHandover] = stateHandover

	ss.hd = NewHandover(nil, t.Encs, t.local.Policy(), t.local.Nodes(), ss.suffrage)

	old := t.newChannel("https://old")

	ss.hd.checkDuplicatedNodeFunc = func() (network.Channel, network.NodeInfo, error) { return old, network.NodeInfoV0{}, nil }

	stopch := make(chan error)
	go func() {
		stopch <- ss.Start()
	}()

	{
		ticker := time.NewTicker(time.Millisecond * 300)
		for range ticker.C {
			if ss.underHandover() {
				break
			}
		}

		ticker.Stop()
		t.T().Log("under handover")
	}

	sctx := NewStateSwitchContext(ss.State(), base.StateConsensus)

	t.NoError(ss.switchState(sctx))
	t.Equal(base.StateSyncing, ss.State())
}

func (t *testStates) TestSwitchingStateToHandoverUnderHandoverHandoverReady() {
	ss := t.newStates()
	defer func() {
		_ = ss.Stop()
	}()

	statech := make(chan struct{})
	badstatech := make(chan struct{})

	stateConsensus := NewBaseState(base.StateConsensus)
	stateConsensus.SetEnterFunc(func(sctx StateSwitchContext) (func() error, error) {
		badstatech <- struct{}{}
		return nil, nil
	})

	stateHandover := NewBaseState(base.StateHandover)
	stateHandover.SetEnterFunc(func(sctx StateSwitchContext) (func() error, error) {
		statech <- struct{}{}
		return nil, nil
	})

	ss.states[base.StateConsensus] = stateConsensus
	ss.states[base.StateHandover] = stateHandover

	ss.hd = NewHandover(nil, t.Encs, t.local.Policy(), t.local.Nodes(), ss.suffrage)

	old := t.newChannel("https://old")

	ss.hd.checkDuplicatedNodeFunc = func() (network.Channel, network.NodeInfo, error) { return old, network.NodeInfoV0{}, nil }

	stopch := make(chan error)
	go func() {
		stopch <- ss.Start()
	}()

	{
		ticker := time.NewTicker(time.Millisecond * 300)
		for range ticker.C {
			if ss.underHandover() {
				break
			}
		}

		ticker.Stop()
		t.T().Log("under handover")
		_ = ss.hd.st.setIsReady(true)
		t.T().Log("handover is ready")
	}

	sctx := NewStateSwitchContext(ss.State(), base.StateConsensus)
	t.NoError(ss.SwitchState(sctx))

	select {
	case err := <-stopch:
		t.NoError(fmt.Errorf("stopped: %w", err))
	case <-badstatech:
		t.NoError(fmt.Errorf("failed to move handover; not consensus"))
	case <-statech:
		t.T().Log("handover entered")
	}
}

func (t *testStates) TestSwitchingStateFromHandoverToConsensusUnderHandoverNotHandoverReady() {
	ss := t.newStates()
	defer func() {
		_ = ss.Stop()
	}()

	statech := make(chan struct{}, 1)
	badstatech := make(chan struct{})

	stateConsensus := NewBaseState(base.StateConsensus)
	stateConsensus.SetEnterFunc(func(sctx StateSwitchContext) (func() error, error) {
		statech <- struct{}{}
		return nil, nil
	})

	stateHandover := NewBaseState(base.StateHandover)
	stateHandover.SetEnterFunc(func(sctx StateSwitchContext) (func() error, error) {
		badstatech <- struct{}{}
		return nil, nil
	})

	exitch := make(chan struct{}, 1)
	stateHandover.SetExitFunc(func(sctx StateSwitchContext) (func() error, error) {
		exitch <- struct{}{}

		return nil, nil
	})

	ss.states[base.StateConsensus] = stateConsensus
	ss.states[base.StateHandover] = stateHandover

	ss.hd = NewHandover(nil, t.Encs, t.local.Policy(), t.local.Nodes(), ss.suffrage)

	old := t.newChannel("https://old")

	ss.hd.checkDuplicatedNodeFunc = func() (network.Channel, network.NodeInfo, error) { return old, network.NodeInfoV0{}, nil }

	stopch := make(chan error)
	go func() {
		stopch <- ss.Start()
	}()

	{
		ticker := time.NewTicker(time.Millisecond * 300)
		for range ticker.C {
			if ss.underHandover() {
				break
			}
		}

		ticker.Stop()
		t.T().Log("under handover")
	}

	ss.setState(base.StateHandover)
	sctx := NewStateSwitchContext(base.StateHandover, base.StateConsensus)
	t.NoError(ss.switchState(sctx))

	select {
	case err := <-stopch:
		t.NoError(fmt.Errorf("stopped: %w", err))
	case <-badstatech:
		t.NoError(fmt.Errorf("failed to move consensus; not handover"))
	case <-statech:
		t.T().Log("consensus entered")
	}

	<-exitch
}

func (t *testStates) TestStartHandover() {
	ss := t.newStates()
	defer func() {
		_ = ss.Stop()
	}()

	ss.hd = NewHandover(nil, t.Encs, t.local.Policy(), t.local.Nodes(), ss.suffrage)

	ss.state = base.StateStopped
	t.Run("state in stopped", func() {
		err := ss.StartHandover()
		t.NotNil(err)
		t.Contains(err.Error(), "node can not start handover")
	})

	ss.state = base.StateBroken
	t.Run("state in broken", func() {
		err := ss.StartHandover()
		t.NotNil(err)
		t.Contains(err.Error(), "node can not start handover")
	})

	ss.state = base.StateHandover
	t.Run("state in handover", func() {
		err := ss.StartHandover()
		t.NotNil(err)
		t.Contains(err.Error(), "node is already in handover")
	})

	ss.state = base.StateConsensus
	t.Run("state in consensus", func() {
		err := ss.StartHandover()
		t.NotNil(err)
		t.Contains(err.Error(), "node is already in consensus")
	})

	ss.state = base.StateJoining

	var old network.Channel
	ss.hd.checkDuplicatedNodeFunc = func() (network.Channel, network.NodeInfo, error) { return old, nil, nil }

	t.Run("handover not under handover", func() {
		err := ss.StartHandover()
		t.NotNil(err)
		t.Contains(err.Error(), "not under handover")
	})

	old = t.newChannel("https://old")
	ss.hd.st.setUnderHandover(true)
	ss.hd.st.setOldNode(old)
	t.Run("handover under handover and is not ready", func() {
		t.NoError(ss.StartHandover())
		t.True(ss.hd.IsReady())
	})

	t.Run("handover under handover, but is already ready", func() {
		err := ss.StartHandover()
		t.NotNil(err)
		t.Contains(err.Error(), "handover was already ready")
	})
}

func (t *testStates) TestEndHandover() {
	ss := t.newStates()
	defer func() {
		_ = ss.Stop()
	}()

	var newnode *isaac.Local
	{
		i := t.Locals(1)
		newnode = i[0]
	}

	ss.joinDiscoveryFunc = func(int, chan error) error {
		return nil
	}

	ss.dis = states.NewTestDiscoveryJoiner()
	ss.hd = NewHandover(nil, t.Encs, t.local.Policy(), t.local.Nodes(), ss.suffrage)

	ss.hd.checkDuplicatedNodeFunc = func() (network.Channel, network.NodeInfo, error) { return newnode.Channel(), network.NodeInfoV0{}, nil }

	leavech := make(chan struct{}, 1)
	ss.dis.SetLeaveFunc(func(time.Duration) error {
		leavech <- struct{}{}

		return nil
	})

	enterch := make(chan StateSwitchContext, 1)
	exitch := make(chan StateSwitchContext, 1)

	stateConsensus := NewBaseState(base.StateConsensus)
	stateConsensus.SetExitFunc(func(sctx StateSwitchContext) (func() error, error) {
		exitch <- sctx

		return nil, nil
	})
	stateSyncing := NewBaseState(base.StateSyncing)
	stateSyncing.SetEnterFunc(func(sctx StateSwitchContext) (func() error, error) {
		enterch <- sctx

		return nil, nil
	})
	ss.states[base.StateConsensus] = stateConsensus
	ss.states[base.StateSyncing] = stateSyncing
	ss.state = base.StateConsensus // NOTE set current state to consensus

	stopch := make(chan error)
	go func() {
		stopch <- ss.Start()
	}()

	ss.dis.SetLeaveFunc(func(time.Duration) error {
		return errors.Errorf("failed to leave discovery")
	})

	newci, err := network.NewHTTPConnInfoFromString("https://newnode", true)
	t.NoError(err)

	t.Run("failed to leave", func() {
		ss.dis.SetJoined(true)
		err := ss.EndHandover(newci)
		t.NotNil(err)
		t.Contains(err.Error(), "failed to leave discovery")
	})

	ss.dis.SetLeaveFunc(func(time.Duration) error {
		leavech <- struct{}{}

		return nil
	})

	t.Run("moves to syncing state", func() {
		ss.dis.SetJoined(true)
		t.NoError(ss.EndHandover(newci))

		select {
		case err := <-stopch:
			t.NoError(fmt.Errorf("stopped: %w", err))
		case <-time.After(time.Second * 3):
			t.NoError(errors.Errorf("timeout to finish end handover"))
		case <-leavech:
		}

		select {
		case err := <-stopch:
			t.NoError(fmt.Errorf("stopped: %w", err))
		case <-time.After(time.Second * 3):
			t.NoError(errors.Errorf("timeout to wait new state"))
		case sctx := <-exitch:
			t.Equal(base.StateConsensus, sctx.FromState())
			t.Equal(base.StateSyncing, sctx.ToState())
		}

		select {
		case err := <-stopch:
			t.NoError(fmt.Errorf("stopped: %w", err))
		case <-time.After(time.Second * 3):
			t.NoError(errors.Errorf("timeout to wait new state"))
		case sctx := <-enterch:
			t.Equal(base.StateConsensus, sctx.FromState())
			t.Equal(base.StateSyncing, sctx.ToState())
		}
	})

	t.Run("operation seal passthrough", func() {
		op, err := operation.NewKVOperation(t.local.Node().Privatekey(), util.UUID().Bytes(), util.UUID().String(), []byte(util.UUID().String()), nil)
		t.NoError(err)
		opsl, err := operation.NewBaseSeal(t.local.Node().Privatekey(), []operation.Operation{op}, t.local.Policy().NetworkID())
		t.NoError(err)

		go func() {
			t.local.Nodes().Passthroughs(context.Background(), network.NewPassthroughedSealFromConnInfo(opsl, nil), func(sl seal.Seal, ch network.Channel) {
				err := ch.SendSeal(context.Background(), nil, sl)
				if err != nil {
					panic(err)
				}
			})
		}()

		// ib := t.NewINITBallot(t.local, base.Round(0), nil)
		rsl := <-newnode.Channel().(*channetwork.Channel).ReceiveSeal()
		t.True(opsl.Hash().Equal(rsl.Hash()))
	})

	t.Run("ballot not passthrough", func() {
		sl := t.NewINITBallot(t.local, base.Round(0), nil)

		go func() {
			t.local.Nodes().Passthroughs(context.Background(), network.NewPassthroughedSealFromConnInfo(sl, nil), func(sl seal.Seal, ch network.Channel) {
				t.NoError(ch.SendSeal(context.Background(), nil, sl))
			})
		}()

		select {
		case <-time.After(time.Second * 3):
		case <-newnode.Channel().(*channetwork.Channel).ReceiveSeal():
			t.NoError(errors.Errorf("ballot will not passthrough after end handover"))
		}
	})
}

func TestStates(t *testing.T) {
	suite.Run(t, new(testStates))
}
