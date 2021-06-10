package basicstates

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testStateJoining struct {
	baseTestState
	local  *isaac.Local
	remote *isaac.Local
}

func (t *testStateJoining) SetupTest() {
	ls := t.Locals(2)
	t.local, t.remote = ls[0], ls[1]
}

func (t *testStateJoining) newState(local *isaac.Local, suffrage base.Suffrage, ballotbox *isaac.Ballotbox) (*JoiningState, func()) {
	st := NewJoiningState(local.Node(), local.Database(), local.Policy(), suffrage, ballotbox)

	return st, func() {
		f, err := st.Exit(NewStateSwitchContext(base.StateJoining, base.StateStopped))
		t.NoError(err)
		_ = f()
	}
}

func (t *testStateJoining) TestBroadcastingINITBallotInStandalone() {
	_, _ = t.local.Policy().SetIntervalBroadcastingINITBallot(time.Millisecond * 30)

	suffrage := t.Suffrage(t.local)
	st, done := t.newState(t.local, suffrage, t.Ballotbox(suffrage, t.local.Policy()))
	defer done()

	timers := localtime.NewTimers([]localtime.TimerID{
		TimerIDBroadcastJoingingINITBallot,
		TimerIDBroadcastINITBallot,
	}, false)
	st.SetTimers(timers)

	sealch := make(chan seal.Seal, 1)
	receivedTime := util.NewLockedItem(nil)
	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		receivedTime.Set(time.Now())
		sealch <- sl

		return nil
	})

	f, err := st.Enter(NewStateSwitchContext(base.StateBooting, base.StateJoining))
	t.NoError(err)
	t.NoError(f())

	started := time.Now()
	wait := t.local.Policy().IntervalBroadcastingINITBallot() * 7

	var received seal.Seal
	select {
	case <-time.After(wait):
	case received = <-sealch:
	}

	t.NotNil(received)

	t.Implements((*seal.Seal)(nil), received)
	t.IsType(ballot.INITV0{}, received)

	t.NotNil(receivedTime.Value())
	t.True(receivedTime.Value().(time.Time).Sub(started) < t.local.Policy().IntervalBroadcastingINITBallot()*3)

	ballot := received.(ballot.INITV0)

	t.NoError(ballot.IsValid(t.local.Policy().NetworkID()))

	manifest := t.LastManifest(t.local.Database())

	t.True(t.local.Node().Publickey().Equal(ballot.Signer()))
	t.Equal(base.StageINIT, ballot.Stage())
	t.Equal(manifest.Height()+1, ballot.Height())
	t.Equal(base.Round(0), ballot.Round())
	t.True(t.local.Node().Address().Equal(ballot.Node()))

	t.True(manifest.Hash().Equal(ballot.PreviousBlock()))
}

func (t *testStateJoining) TestBroadcastingINITBallotWithoutACCEPTVoteproof() {
	_, _ = t.local.Policy().SetIntervalBroadcastingINITBallot(time.Millisecond * 30)

	suffrage := t.Suffrage(t.local, t.remote)
	st, done := t.newState(t.local, suffrage, t.Ballotbox(suffrage, t.local.Policy()))
	defer done()

	timers := localtime.NewTimers([]localtime.TimerID{
		TimerIDBroadcastJoingingINITBallot,
		TimerIDBroadcastINITBallot,
	}, false)
	st.SetTimers(timers)

	sealch := make(chan seal.Seal, 1)
	receivedTime := util.NewLockedItem(nil)
	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		receivedTime.Set(time.Now())
		sealch <- sl

		return nil
	})

	f, err := st.Enter(NewStateSwitchContext(base.StateBooting, base.StateJoining))
	t.NoError(err)
	t.NoError(f())

	started := time.Now()
	wait := t.local.Policy().IntervalBroadcastingINITBallot() * 7

	var received seal.Seal
	select {
	case <-time.After(wait):
	case received = <-sealch:
	}

	t.NotNil(received)

	t.Implements((*seal.Seal)(nil), received)
	t.IsType(ballot.INITV0{}, received)

	t.NotNil(receivedTime.Value())
	t.True(receivedTime.Value().(time.Time).Sub(started) > t.local.Policy().IntervalBroadcastingINITBallot()*3)

	ballot := received.(ballot.INITV0)

	t.NoError(ballot.IsValid(t.local.Policy().NetworkID()))

	manifest := t.LastManifest(t.local.Database())

	t.True(t.local.Node().Publickey().Equal(ballot.Signer()))
	t.Equal(base.StageINIT, ballot.Stage())
	t.Equal(manifest.Height()+1, ballot.Height())
	t.Equal(base.Round(0), ballot.Round())
	t.True(t.local.Node().Address().Equal(ballot.Node()))

	t.True(manifest.Hash().Equal(ballot.PreviousBlock()))
}

// TestBroadcastingINITBallotWithACCEPTVoteproof tests, with accept voteproof
// states waits new voteproof until timeout. If no voteproof after timeout,
// tries to broadcast new INIT ballot from initial accept voteproof or voteproof
// of collected ballot.
func (t *testStateJoining) TestBroadcastingINITBallotWithACCEPTVoteproof() {
	_, _ = t.local.Policy().SetIntervalBroadcastingINITBallot(time.Millisecond * 30)

	suffrage := t.Suffrage(t.local, t.remote)
	st, done := t.newState(t.local, suffrage, t.Ballotbox(suffrage, t.local.Policy()))
	defer done()

	timers := localtime.NewTimers([]localtime.TimerID{
		TimerIDBroadcastJoingingINITBallot,
		TimerIDBroadcastINITBallot,
	}, false)
	st.SetTimers(timers)

	sealch := make(chan seal.Seal, 1)
	receivedTime := util.NewLockedItem(nil)
	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		receivedTime.Set(time.Now())
		sealch <- sl

		return nil
	})

	lastAcceptVoteproof := t.local.Database().LastVoteproof(base.StageACCEPT)
	t.NotNil(lastAcceptVoteproof)

	f, err := st.Enter(NewStateSwitchContext(base.StateBooting, base.StateJoining).SetVoteproof(lastAcceptVoteproof))
	t.NoError(err)
	t.NoError(f())

	started := time.Now()
	wait := t.local.Policy().IntervalBroadcastingINITBallot() * 7

	var received seal.Seal
	select {
	case <-time.After(wait):
	case received = <-sealch:
	}

	t.NotNil(received)

	t.Implements((*seal.Seal)(nil), received)
	t.IsType(ballot.INITV0{}, received)

	t.NotNil(receivedTime.Value())
	t.True(receivedTime.Value().(time.Time).Sub(started) > t.local.Policy().IntervalBroadcastingINITBallot()*3)

	ballot := received.(ballot.INITV0)

	t.NoError(ballot.IsValid(t.local.Policy().NetworkID()))

	manifest := t.LastManifest(t.local.Database())

	t.True(t.local.Node().Publickey().Equal(ballot.Signer()))
	t.Equal(base.StageINIT, ballot.Stage())
	t.Equal(manifest.Height()+1, ballot.Height())
	t.Equal(base.Round(0), ballot.Round())
	t.True(t.local.Node().Address().Equal(ballot.Node()))

	t.True(manifest.Hash().Equal(ballot.PreviousBlock()))
}

func (t *testStateJoining) TestTimerStopAfterExit() {
	_, _ = t.local.Policy().SetIntervalBroadcastingINITBallot(time.Millisecond * 30)

	suffrage := t.Suffrage(t.local)
	st, done := t.newState(t.local, suffrage, t.Ballotbox(suffrage, t.local.Policy()))
	defer done()

	timers := localtime.NewTimers([]localtime.TimerID{
		TimerIDBroadcastJoingingINITBallot,
		TimerIDBroadcastINITBallot,
	}, false)
	st.SetTimers(timers)
	st.SetBroadcastSealsFunc(func(seal.Seal, bool) error { return nil })

	lastAcceptVoteproof := t.local.Database().LastVoteproof(base.StageACCEPT)
	t.NotNil(lastAcceptVoteproof)

	f, err := st.Enter(NewStateSwitchContext(base.StateBooting, base.StateJoining).SetVoteproof(lastAcceptVoteproof))
	t.NoError(err)
	t.NoError(f())

	<-time.After(t.local.Policy().IntervalBroadcastingINITBallot() * 7)

	t.Equal([]localtime.TimerID{TimerIDBroadcastJoingingINITBallot}, timers.Started())

	f, err = st.Exit(NewStateSwitchContext(base.StateJoining, base.StateConsensus))
	t.NoError(err)
	t.NoError(f())

	t.Empty(timers.Started())
}

func (t *testStateJoining) TestCheckBallotboxWithINITBallot() {
	_, _ = t.local.Policy().SetIntervalBroadcastingINITBallot(time.Millisecond * 30)

	suffrage := t.Suffrage(t.local, t.remote)
	ballotbox := t.Ballotbox(suffrage, t.local.Policy())

	st, done := t.newState(t.local, suffrage, ballotbox)
	defer done()

	timers := localtime.NewTimers([]localtime.TimerID{
		TimerIDBroadcastJoingingINITBallot,
		TimerIDBroadcastINITBallot,
	}, false)
	st.SetTimers(timers)

	st.SetBroadcastSealsFunc(func(seal.Seal, bool) error {
		return nil
	})

	vpch := make(chan base.Voteproof)
	st.SetNewVoteproofFunc(func(voteproof base.Voteproof) {
		vpch <- voteproof
	})

	lastINITVoteproof := t.local.Database().LastVoteproof(base.StageINIT)
	t.NotNil(lastINITVoteproof)

	st.SetLastVoteproofFuncs(func() base.Voteproof {
		return lastINITVoteproof
	}, func() base.Voteproof {
		return lastINITVoteproof
	}, nil)

	lastACCEPTVoteproof := t.local.Database().LastVoteproof(base.StageACCEPT)
	t.NotNil(lastACCEPTVoteproof)

	initFact := ballot.NewINITV0(
		t.local.Node().Address(),
		lastACCEPTVoteproof.Height()+1,
		base.Round(0),
		valuehash.RandomSHA256(),
		nil,
		nil,
	).Fact().(ballot.INITFactV0)

	initVoteproof, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	acceptBallot := ballot.NewACCEPTV0(
		t.local.Node().Address(),
		initVoteproof.Height(),
		initVoteproof.Round(),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		initVoteproof,
	)
	t.NoError(acceptBallot.Sign(t.local.Node().Privatekey(), t.local.Policy().NetworkID()))

	f, err := st.Enter(NewStateSwitchContext(base.StateBooting, base.StateJoining).SetVoteproof(lastACCEPTVoteproof))
	t.NoError(err)
	t.NoError(f())

	// send init voteproof by accept ballot
	_, err = ballotbox.Vote(acceptBallot)
	t.NoError(err)

	wait := t.local.Policy().IntervalBroadcastingINITBallot() * 7

	var received base.Voteproof
	select {
	case <-time.After(wait):
	case voteproof := <-vpch:
		received = voteproof
	}

	t.NotNil(received)
	t.Equal(initVoteproof.Bytes(), received.Bytes())
}

func (t *testStateJoining) TestNewINITVoteproof() {
	_, _ = t.local.Policy().SetIntervalBroadcastingINITBallot(time.Millisecond * 30)

	suffrage := t.Suffrage(t.local)
	ballotbox := t.Ballotbox(suffrage, t.local.Policy())

	st, done := t.newState(t.local, suffrage, ballotbox)
	defer done()

	timers := localtime.NewTimers([]localtime.TimerID{
		TimerIDBroadcastJoingingINITBallot,
		TimerIDBroadcastINITBallot,
	}, false)
	st.SetTimers(timers)

	st.SetBroadcastSealsFunc(func(seal.Seal, bool) error {
		return nil
	})

	statech := make(chan StateSwitchContext)
	st.SetStateSwitchFunc(func(sctx StateSwitchContext) error {
		statech <- sctx

		return nil
	})

	lastINITVoteproof := t.local.Database().LastVoteproof(base.StageINIT)
	t.NotNil(lastINITVoteproof)

	st.SetLastVoteproofFuncs(func() base.Voteproof {
		return lastINITVoteproof
	}, func() base.Voteproof {
		return lastINITVoteproof
	}, nil)

	lastACCEPTVoteproof := t.local.Database().LastVoteproof(base.StageACCEPT)
	t.NotNil(lastACCEPTVoteproof)

	f, err := st.Enter(NewStateSwitchContext(base.StateBooting, base.StateJoining).SetVoteproof(lastACCEPTVoteproof))
	t.NoError(err)
	t.NoError(f())

	initFact := ballot.NewINITV0(
		t.local.Node().Address(),
		lastACCEPTVoteproof.Height()+1,
		base.Round(0),
		valuehash.RandomSHA256(),
		nil,
		nil,
	).Fact().(ballot.INITFactV0)

	newINITVoteproof, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	// before timeout, inject new init voteproof
	<-time.After(t.local.Policy().IntervalBroadcastingINITBallot() * 2)
	err = st.ProcessVoteproof(newINITVoteproof)

	var received StateSwitchContext
	t.True(xerrors.As(err, &received))

	t.Equal(base.StateJoining, received.FromState())
	t.Equal(base.StateConsensus, received.ToState())
	t.NotNil(received.Voteproof())
	t.Equal(newINITVoteproof.Bytes(), received.Voteproof().Bytes())
}

func TestStateJoining(t *testing.T) {
	suite.Run(t, new(testStateJoining))
}
