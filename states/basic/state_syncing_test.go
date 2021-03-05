package basicstates

import (
	"testing"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/stretchr/testify/suite"
)

type testStateSyncing struct {
	baseTestState
	local  *isaac.Local
	remote *isaac.Local
}

func (t *testStateSyncing) SetupTest() {
	ls := t.Locals(2)
	t.local, t.remote = ls[0], ls[1]
}

func (t *testStateSyncing) newState(local *isaac.Local) (*SyncingState, func()) {
	st := NewSyncingState(local.Storage(), local.BlockFS(), local.Policy(), local.Nodes())

	return st, func() {
		f, err := st.Exit(NewStateSwitchContext(base.StateSyncing, base.StateStopped))
		t.NoError(err)
		_ = f()
	}
}

func (t *testStateSyncing) TestINITMovesToConsensus() {
	st, done := t.newState(t.local)
	defer done()

	timers := localtime.NewTimers([]localtime.TimerID{
		TimerIDSyncingWaitVoteproof,
	}, false)
	st.SetTimers(timers)

	lastINITVoteproof, found, err := t.remote.BlockFS().LastVoteproof(base.StageINIT)
	t.NoError(err)
	t.True(found)

	var voteproof base.Voteproof
	{
		b := t.NewINITBallot(t.remote, base.Round(0), lastINITVoteproof)

		vp, err := t.NewVoteproof(b.Stage(), b.INITBallotFactV0, t.remote)
		t.NoError(err)

		voteproof = vp
	}

	f, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateSyncing).SetVoteproof(voteproof))
	t.NoError(err)
	err = f()

	var sctx StateSwitchContext
	t.True(xerrors.As(err, &sctx))
	t.Equal(base.StateSyncing, sctx.FromState())
	t.Equal(base.StateConsensus, sctx.ToState())
	t.Equal(voteproof.Bytes(), sctx.Voteproof().Bytes())
}

func (t *testStateSyncing) TestWaitMovesToJoining() {
	t.local.Policy().SetTimeoutWaitingProposal(time.Millisecond * 10)

	st, done := t.newState(t.local)
	defer done()

	st.waitVoteproofTimeout = time.Millisecond * 10

	timers := localtime.NewTimers([]localtime.TimerID{
		TimerIDSyncingWaitVoteproof,
	}, false)
	st.SetTimers(timers)

	lastINITVoteproof, found, err := t.remote.BlockFS().LastVoteproof(base.StageINIT)
	t.NoError(err)
	t.True(found)

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
		t.NoError(xerrors.Errorf("timeout to wait to move joining state"))
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

	baseBlock := t.LastManifest(local.Storage())
	target := baseBlock.Height() + 5
	t.GenerateBlocks([]*isaac.Local{rn0, rn1, rn2}, target)

	st, done := t.newState(local)
	defer done()

	st.waitVoteproofTimeout = time.Minute * 50

	timers := localtime.NewTimers([]localtime.TimerID{
		TimerIDSyncingWaitVoteproof,
	}, false)
	st.SetTimers(timers)

	livpch := make(chan base.Voteproof)
	st.SetLastVoteproofFuncs(func() base.Voteproof {
		return nil
	}, func() base.Voteproof {
		return nil
	}, func(voteproof base.Voteproof) {
		livpch <- voteproof
	})

	savedblocksch := make(chan []block.Block)
	st.SetNewBlocksFunc(func(blks []block.Block) error {
		savedblocksch <- blks
		return nil
	})

	var voteproof base.Voteproof
	{
		b := t.NewINITBallot(rn0, base.Round(0), nil)

		vp, err := t.NewVoteproof(b.Stage(), b.INITBallotFactV0, rn0)
		t.NoError(err)

		voteproof = vp
	}

	f, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateSyncing).SetVoteproof(voteproof))
	t.NoError(err)
	t.NoError(f())

	finishedChan := make(chan struct{})
	go func() {
		for {
			b, found, err := local.Storage().LastManifest()
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
		t.NoError(xerrors.Errorf("timeout to wait to be finished"))

		return
	case <-finishedChan:
		break
	}

	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("timeout to wait to set last init voteproof"))

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
}

func TestStateSyncing(t *testing.T) {
	suite.Run(t, new(testStateSyncing))
}
