package basicstates

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	channetwork "github.com/spikeekips/mitum/network/gochan"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testStates struct {
	baseTestState
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
	)
	t.NoError(err)
	t.NotNil(ss)

	livp := t.local.Database().LastVoteproof(base.StageINIT)
	t.NotNil(livp)

	sslivp := ss.LastINITVoteproof()
	t.Equal(livp.Bytes(), sslivp.Bytes())

	t.Equal(base.StateStopped, ss.State())
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
	)
	t.NoError(err)
	t.NotNil(ss)

	return ss
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
	ibr := t.NewINITBallot(t.remote, ibl.Round(), nil)

	t.NoError(ss.NewSeal(ibl))
	t.NoError(ss.NewSeal(ibr))

	select {
	case err := <-stopch:
		t.NoError(err)
	case voteproof := <-gotvoteproofch:
		lib := ss.ballotbox.LatestBallot()
		t.NotNil(lib)

		t.NotNil(voteproof)

		t.Equal(base.StageINIT, voteproof.Stage())
		t.Equal(ibl.Height(), voteproof.Height())
		t.Equal(ibl.Round(), voteproof.Round())
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
		t.NoError(err)
	case nsctx := <-statech:
		t.Equal(sctx.FromState(), nsctx.FromState())
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
		t.NoError(err)
	case nsctx := <-statech:
		t.Equal(sctx.FromState(), nsctx.FromState())
		t.Equal(sctx.ToState(), nsctx.ToState())
	}

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	ss.NewVoteproof(ivp)

	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("timeout to wait voteproof thru consensus state"))
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
		t.NoError(err)
	case nsctx := <-statech:
		t.Equal(sctx.FromState(), nsctx.FromState())
		t.Equal(sctx.ToState(), nsctx.ToState())
	}

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	t.NoError(ss.SwitchState(NewStateSwitchContext(base.StateConsensus, base.StateConsensus).SetVoteproof(ivp)))

	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("timeout to wait voteproof thru consensus state"))
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
		t.NoError(err)
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
		t.NoError(xerrors.Errorf("timeout to wait voteproof thru consensus state"))
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
		return nil, xerrors.Errorf("born to be killed")
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
		t.NoError(xerrors.Errorf("timeout to wait to switch state"))
	case err := <-stopch:
		t.NoError(err)
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

		return nil, xerrors.Errorf("impossible entering")
	})

	stateJoining := NewBaseState(base.StateJoining)
	stateJoining.SetEnterFunc(func(StateSwitchContext) (func() error, error) {
		return nil, xerrors.Errorf("born to be killed")
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
		t.NoError(xerrors.Errorf("timeout to wait states to be stopped"))
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
			return xerrors.Errorf("exit error")
		}, nil
	})

	stateJoining := NewBaseState(base.StateJoining)
	stateJoining.SetEnterFunc(func(StateSwitchContext) (func() error, error) {
		return func() error {
			return xerrors.Errorf("enter error")
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
		t.NoError(xerrors.Errorf("waited broadcasted seal, but nothing"))
	case rsl := <-remotech.ReceiveSeal():
		t.True(sl.Hash().Equal(rsl.Hash()))
	}
}

func TestStates(t *testing.T) {
	suite.Run(t, new(testStates))
}
