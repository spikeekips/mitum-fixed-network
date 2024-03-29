package basicstates

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/stretchr/testify/suite"
)

type testStateSyncing struct {
	baseTestState
}

func (t *testStateSyncing) newState(local *isaac.Local, suffrage base.Suffrage) (*SyncingState, func()) {
	st := NewSyncingState(local.Database(), local.Blockdata(), local.Policy(), local.Nodes(), suffrage)

	return st, func() {
		f, err := st.Exit(NewStateSwitchContext(base.StateSyncing, base.StateStopped))
		t.NoError(err)
		_ = f()
	}
}

func (t *testStateSyncing) TestINITMovesToConsensus() {
	st, done := t.newState(t.local, t.Suffrage(t.local, t.remote))
	defer done()

	st.SetLastVoteproofFuncs(func() base.Voteproof {
		return nil
	}, func() base.Voteproof {
		return nil
	}, func(voteproof base.Voteproof) bool {
		return true
	})

	timers := localtime.NewTimers([]localtime.TimerID{
		TimerIDSyncingWaitVoteproof,
	}, false)
	st.SetTimers(timers)

	lastINITVoteproof := t.remote.Database().LastVoteproof(base.StageINIT)
	t.NotNil(lastINITVoteproof)

	var voteproof base.Voteproof
	{
		b := t.NewINITBallot(t.remote, base.Round(0), lastINITVoteproof)

		vp, err := t.NewVoteproof(b.Fact().Stage(), b.Fact(), t.remote)
		t.NoError(err)

		voteproof = vp
	}

	f, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateSyncing).SetVoteproof(voteproof))
	t.NoError(err)
	err = f()

	var sctx StateSwitchContext
	t.True(errors.As(err, &sctx))
	t.Equal(base.StateSyncing, sctx.FromState())
	t.Equal(base.StateConsensus, sctx.ToState())
	t.Equal(voteproof.Bytes(), sctx.Voteproof().Bytes())
}

func (t *testStateSyncing) TestWaitMovesToJoining() {
	t.local.Policy().SetTimeoutWaitingProposal(time.Millisecond * 10)

	st, done := t.newState(t.local, t.Suffrage(t.local, t.remote))
	defer done()

	st.waitVoteproofTimeout = time.Millisecond * 10
	st.SetLastVoteproofFuncs(func() base.Voteproof {
		return nil
	}, func() base.Voteproof {
		return nil
	}, func(voteproof base.Voteproof) bool {
		return true
	})

	timers := localtime.NewTimers([]localtime.TimerID{
		TimerIDSyncingWaitVoteproof,
	}, false)
	st.SetTimers(timers)

	lastINITVoteproof := t.remote.Database().LastVoteproof(base.StageINIT)
	t.NotNil(lastINITVoteproof)

	statech := make(chan StateSwitchContext)
	st.SetStateSwitchFunc(func(sctx StateSwitchContext) error {
		statech <- sctx

		return nil
	})

	f, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateSyncing).SetVoteproof(lastINITVoteproof))
	t.NoError(err)
	t.NoError(f())

	st.whenFinished(base.NilHeight)

	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("timeout to wait to move joining state"))
	case sctx := <-statech:
		t.Equal(base.StateSyncing, sctx.FromState())
		t.Equal(base.StateJoining, sctx.ToState())
		t.Nil(sctx.Voteproof())
	}
}

func (t *testStateSyncing) TestSyncingHandlerFromVoteproof() {
	ls := t.Locals(4)
	local, rn0, rn1, rn2 := ls[0], ls[1], ls[2], ls[3]

	t.SetupNodes(local, []*isaac.Local{rn0, rn1, rn2})

	baseBlock := t.LastManifest(local.Database())
	target := baseBlock.Height() + 5
	t.GenerateBlocks([]*isaac.Local{rn0, rn1, rn2}, target)

	st, done := t.newState(local, t.Suffrage(local, rn0, rn1, rn2))
	defer done()

	st.waitVoteproofTimeout = time.Minute * 50

	timers := localtime.NewTimers([]localtime.TimerID{
		TimerIDSyncingWaitVoteproof,
	}, false)
	st.SetTimers(timers)

	livpch := make(chan base.Voteproof, 2)
	st.SetLastVoteproofFuncs(func() base.Voteproof {
		return nil
	}, func() base.Voteproof {
		return nil
	}, func(voteproof base.Voteproof) bool {
		livpch <- voteproof

		return true
	})

	savedblocksch := make(chan []block.Block)
	st.SetNewBlocksFunc(func(blks []block.Block) error {
		go func() {
			savedblocksch <- blks
		}()

		return nil
	})

	var voteproof base.Voteproof
	{
		b := t.NewINITBallot(rn0, base.Round(0), nil)

		vp, err := t.NewVoteproof(b.Fact().Stage(), b.Fact(), rn0)
		t.NoError(err)

		voteproof = vp
	}

	f, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateSyncing).SetVoteproof(voteproof))
	t.NoError(err)
	t.NoError(f())

	finishedChan := make(chan struct{})
	go func() {
		for {
			b, found, err := local.Database().LastManifest()
			t.NoError(err)
			t.True(found)

			if b.Height() == voteproof.Height()-1 {
				finishedChan <- struct{}{}
				break
			}

			<-time.After(time.Millisecond * 100)
		}
	}()

	select {
	case <-time.After(time.Second * 10):
		t.NoError(errors.Errorf("timeout to wait to be finished"))

		return
	case <-finishedChan:
		break
	}

	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("timeout to wait to set last init voteproof"))

		return
	case vp := <-livpch:
		t.Equal(target, vp.Height())
	}

	var blks []block.Block

end:
	for {
		select {
		case <-time.After(time.Second * 3):
			break end
		case i := <-savedblocksch:
			blks = append(blks, i...)
		}
	}

	t.Equal(int(target-baseBlock.Height()), len(blks))
	t.Equal(baseBlock.Height()+1, blks[0].Height())
	t.Equal(target, blks[len(blks)-1].Height())

	err = st.ProcessVoteproof(voteproof)

	var sctx StateSwitchContext
	t.True(errors.As(err, &sctx))
	t.Equal(base.StateSyncing, sctx.FromState())
	t.Equal(base.StateConsensus, sctx.ToState())
	t.Equal(voteproof.Bytes(), sctx.Voteproof().Bytes())
}

func (t *testStateSyncing) TestNoneSuffrage() {
	st, done := t.newState(t.local, t.Suffrage(t.remote))
	defer done()

	timers := localtime.NewTimers([]localtime.TimerID{
		TimerIDSyncingWaitVoteproof,
	}, false)
	st.SetTimers(timers)

	f, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateSyncing))
	t.NoError(err)
	err = f()
	t.NoError(err)

	t.True(st.canStartNodeInfoChecker())
	t.True(st.nc.IsStarted())
}

func (t *testStateSyncing) readyToFinish(local *isaac.Local, suffrage base.Suffrage, others ...*isaac.Local) (*SyncingState, chan bool) {
	t.SetupNodes(local, others)

	baseBlock := t.LastManifest(local.Database())
	target := baseBlock.Height() + 5
	t.GenerateBlocks(others, target)

	st, done := t.newState(local, suffrage)
	defer done()

	stt := t.newStates(local, suffrage, st)
	stt.hd = NewHandover(nil, t.Encs, local.Policy(), local.Nodes(), suffrage)
	st.States = stt

	st.waitVoteproofTimeout = time.Minute * 50

	timers := localtime.NewTimers([]localtime.TimerID{
		TimerIDSyncingWaitVoteproof,
	}, false)
	st.SetTimers(timers)

	var livp base.Voteproof
	st.SetLastVoteproofFuncs(func() base.Voteproof {
		return livp
	}, func() base.Voteproof {
		if livp.Stage() == base.StageINIT {
			return livp
		}

		return nil
	}, func(voteproof base.Voteproof) bool {
		livp = voteproof

		return true
	})

	finishedch := make(chan bool)
	st.SetNewBlocksFunc(func(blks []block.Block) error {
		for _, blk := range blks {
			if blk.Height() == target {
				finishedch <- true
			}
		}

		return nil
	})

	return st, finishedch
}

func (t *testStateSyncing) TestFinishedButNotInSuffrage() {
	suffrage := t.Suffrage(t.remote)
	st, finishedch := t.readyToFinish(t.local, suffrage, t.remote)
	t.False(st.States.underHandover())
	t.False(st.States.isHandoverReady())

	var voteproof base.Voteproof
	{
		b := t.NewINITBallot(t.remote, base.Round(0), nil)

		vp, err := t.NewVoteproof(b.Fact().Stage(), b.Fact(), t.remote)
		t.NoError(err)

		voteproof = vp
	}

	f, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateSyncing).SetVoteproof(voteproof))
	t.NoError(err)
	t.NoError(f())

	select {
	case <-time.After(time.Second * 10):
		t.NoError(errors.Errorf("timeout to wait to be finished"))

		return
	case <-finishedch:
		<-time.After(time.Second)
	}

	t.NoError(st.ProcessVoteproof(voteproof)) // NOTE stay in syncing
}

func (t *testStateSyncing) TestFinishedButUnderhandover() {
	suffrage := t.Suffrage(t.local, t.remote)
	st, finishedch := t.readyToFinish(t.local, suffrage, t.remote)
	st.States.hd.st.setUnderHandover(true)
	t.True(st.States.underHandover())
	t.False(st.States.isHandoverReady())

	var voteproof base.Voteproof
	{
		b := t.NewINITBallot(t.remote, base.Round(0), nil)

		vp, err := t.NewVoteproof(b.Fact().Stage(), b.Fact(), t.remote)
		t.NoError(err)

		voteproof = vp
	}

	f, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateSyncing).SetVoteproof(voteproof))
	t.NoError(err)
	t.NoError(f())

	select {
	case <-time.After(time.Second * 10):
		t.NoError(errors.Errorf("timeout to wait to be finished"))

		return
	case <-finishedch:
		<-time.After(time.Second)
	}

	t.NoError(st.ProcessVoteproof(voteproof)) // NOTE stay in syncing
}

func (t *testStateSyncing) TestFinishedButNotReadyHandover() {
	suffrage := t.Suffrage(t.local, t.remote)
	st, finishedch := t.readyToFinish(t.local, suffrage, t.remote)
	st.States.hd.st.setUnderHandover(true)
	t.True(st.States.underHandover())
	t.False(st.States.isHandoverReady())

	var voteproof base.Voteproof
	{
		b := t.NewINITBallot(t.remote, base.Round(0), nil)

		vp, err := t.NewVoteproof(b.Fact().Stage(), b.Fact(), t.remote)
		t.NoError(err)

		voteproof = vp
	}

	f, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateSyncing).SetVoteproof(voteproof))
	t.NoError(err)
	t.NoError(f())

	select {
	case <-time.After(time.Second * 10):
		t.NoError(errors.Errorf("timeout to wait to be finished"))

		return
	case <-finishedch:
		<-time.After(time.Second)
	}

	t.NoError(st.ProcessVoteproof(voteproof)) // NOTE stay in syncing
}

func (t *testStateSyncing) TestFinishedUnderhandoverAndReady() {
	suffrage := t.Suffrage(t.local, t.remote)
	st, finishedch := t.readyToFinish(t.local, suffrage, t.remote)
	st.States.hd.st.setUnderHandover(true)
	st.States.hd.st.setIsReady(true)
	t.True(st.States.underHandover())
	t.True(st.States.isHandoverReady())

	var voteproof base.Voteproof
	{
		b := t.NewINITBallot(t.remote, base.Round(0), nil)

		vp, err := t.NewVoteproof(b.Fact().Stage(), b.Fact(), t.remote)
		t.NoError(err)

		voteproof = vp
	}

	f, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateSyncing).SetVoteproof(voteproof))
	t.NoError(err)
	t.NoError(f())

	select {
	case <-time.After(time.Second * 10):
		t.NoError(errors.Errorf("timeout to wait to be finished"))

		return
	case <-finishedch:
		<-time.After(time.Second)
	}

	err = st.ProcessVoteproof(voteproof)

	var sctx StateSwitchContext
	t.True(errors.As(err, &sctx))
	t.Equal(base.StateSyncing, sctx.FromState())
	t.Equal(base.StateConsensus, sctx.ToState())
	t.Equal(voteproof.Bytes(), sctx.Voteproof().Bytes())
}

func TestStateSyncing(t *testing.T) {
	suite.Run(t, new(testStateSyncing))
}
