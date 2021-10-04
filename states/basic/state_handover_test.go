package basicstates

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	channetwork "github.com/spikeekips/mitum/network/gochan"
	"github.com/spikeekips/mitum/states"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testStateHandover struct {
	baseTestState
	old *isaac.Local
}

func (t *testStateHandover) SetupTest() {
	t.baseTestState.SetupTest()

	ls := t.Locals(1)
	t.old = ls[0]
}

func (t *testStateHandover) newState(suffrage base.Suffrage, pps *prprocessor.Processors) (*HandoverState, func()) {
	if suffrage == nil {
		suffrage = t.Suffrage(t.remote, t.local)
	}

	if pps == nil {
		pps = prprocessor.NewProcessors(isaac.NewDefaultProcessorNewFunc(
			t.local.Database(),
			t.local.BlockData(),
			t.local.Nodes(),
			suffrage,
			nil,
		), nil)

		t.NoError(pps.Initialize())
		t.NoError(pps.Start())
	}

	st := NewHandoverState(
		t.local.Database(),
		t.local.Policy(),
		t.local.Nodes(),
		suffrage,
		pps,
	)

	stt := t.newStates(t.local, suffrage, st)
	stt.dis = states.NewTestDiscoveryJoiner()
	stt.joinDiscoveryFunc = func(int, chan error) error {
		return nil
	}

	timers := localtime.NewTimers([]localtime.TimerID{
		TimerIDBroadcastINITBallot,
		TimerIDBroadcastProposal,
		TimerIDBroadcastACCEPTBallot,
		TimerIDFindProposal,
	}, false)
	st.SetTimers(timers)
	st.States = stt

	hd := NewHandover(t.local.Channel().ConnInfo(), t.Encs, t.local.Policy(), t.local.Nodes(), suffrage)
	_ = hd.st.setUnderHandover(true)

	st.States.hd = hd

	lastINITVoteproof := t.local.Database().LastVoteproof(base.StageINIT)
	t.NotNil(lastINITVoteproof)

	st.SetLastVoteproofFuncs(func() base.Voteproof {
		return lastINITVoteproof
	}, func() base.Voteproof {
		return lastINITVoteproof
	}, nil)

	return st, func() {
		_ = pps.Stop()
		f, err := st.Exit(NewStateSwitchContext(base.StateHandover, base.StateStopped))
		t.NoError(err)
		_ = f()

		_ = timers.Stop()
	}
}

func (t *testStateHandover) nextINITVoteproof(local *isaac.Local, prevINIT, prevACCEPT base.Voteproof, round base.Round, states ...*isaac.Local) base.Voteproof {
	var ib ballot.INITV0
	if prevACCEPT == nil {
		ib = t.NewINITBallot(local, base.Round(0), nil)
	} else {
		fact := prevACCEPT.Facts()[0].(ballot.ACCEPTFactV0)
		ib = ballot.NewINITV0(
			local.Node().Address(),
			prevACCEPT.Height()+1,
			round,
			fact.NewBlock(),
			prevINIT,
			prevACCEPT,
		)
		t.NoError(ib.Sign(local.Node().Privatekey(), local.Policy().NetworkID()))
	}

	vp, err := t.NewVoteproof(base.StageINIT, ib.INITFactV0, states...)
	t.NoError(err)

	return vp
}

func (t *testStateHandover) nextACCEPTVoteproof(local *isaac.Local, pr ballot.Proposal, newBlock valuehash.Hash, states ...*isaac.Local) base.Voteproof {
	ab := t.NewACCEPTBallot(local, pr.Round(), pr.Hash(), newBlock, pr.Voteproof())

	vp, err := t.NewVoteproof(base.StageACCEPT, ab.ACCEPTFactV0, states...)
	t.NoError(err)

	return vp
}

func (t *testStateHandover) checkJoined(st *HandoverState) bool {
	if st.States == nil {
		return false
	}

	ticker := time.NewTicker(time.Millisecond * 100)

	for {
		select {
		case <-time.After(time.Second * 2):
			return false
		case <-ticker.C:
			if st.States.isJoined() {
				return true
			}
		}
	}
}

func (t *testStateHandover) TestNotInSuffrage() {
	suffrage := t.Suffrage(t.remote)

	st := NewHandoverState(nil, nil, t.local.Nodes(), suffrage, nil)
	defer st.Exit(NewStateSwitchContext(base.StateHandover, base.StateStopped))

	hd := NewHandover(nil, t.Encs, t.local.Policy(), t.local.Nodes(), suffrage)
	_ = hd.st.setUnderHandover(true)

	st.States = t.newStates(t.local, suffrage, st)
	st.States.hd = hd

	sctx := NewStateSwitchContext(base.StateBooting, base.StateHandover)

	_, err := st.Enter(sctx)

	var usctx StateSwitchContext
	t.True(errors.As(err, &usctx))

	t.Equal(base.StateHandover, usctx.FromState())
	t.Equal(base.StateConsensus, usctx.ToState())
}

func (t *testStateHandover) TestEmptyOldNode() {
	suffrage := t.Suffrage(t.local)

	st := NewHandoverState(nil, nil, t.local.Nodes(), suffrage, nil)
	defer st.Exit(NewStateSwitchContext(base.StateHandover, base.StateStopped))

	hd := NewHandover(nil, t.Encs, t.local.Policy(), t.local.Nodes(), suffrage)

	st.States = t.newStates(t.local, suffrage, st)
	st.States.hd = hd

	sctx := NewStateSwitchContext(base.StateBooting, base.StateHandover)

	_, err := st.Enter(sctx)

	var usctx StateSwitchContext
	t.True(errors.As(err, &usctx))

	t.Equal(base.StateHandover, usctx.FromState())
	t.Equal(base.StateConsensus, usctx.ToState())
}

func (t *testStateHandover) TestNewProposalBroadcasted() {
	st, done := t.newState(t.Suffrage(t.local, t.remote), nil)
	defer done()

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	sealch := make(chan seal.Seal, 1)
	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		if _, ok := sl.(ballot.Proposal); !ok {
			return nil
		}

		sealch <- sl

		return nil
	})

	t.local.Policy().SetTimeoutWaitingProposal(time.Millisecond * 10)
	f, err := st.Enter(NewStateSwitchContext(base.StateSyncing, base.StateHandover).
		SetVoteproof(ivp),
	)
	t.NoError(err)
	t.NoError(f())

	pr := t.NewProposal(t.old, initFact.Round(), nil, ivp) // new proposal from old
	t.NoError(st.ProcessProposal(pr))

	select {
	case <-time.After(time.Second * 2):
		t.NoError(errors.Errorf("timeout to wait to broadcast new proposal, which is from old"))
	case sl := <-sealch:
		t.Implements((*ballot.Proposal)(nil), sl)
		t.True(sl.Hash().Equal(pr.Hash()))
	}
}

func (t *testStateHandover) TestNewProposalNextAcceptBallotNotBroadcasted() {
	st, done := t.newState(t.Suffrage(t.local, t.remote), nil)
	defer done()

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	sealch := make(chan seal.Seal, 1)
	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		if _, ok := sl.(ballot.ACCEPT); !ok {
			return nil
		}

		sealch <- sl

		return nil
	})

	t.local.Policy().SetTimeoutWaitingProposal(time.Millisecond * 10)
	f, err := st.Enter(NewStateSwitchContext(base.StateSyncing, base.StateHandover).
		SetVoteproof(ivp),
	)
	t.NoError(err)
	t.NoError(f())

	pr := t.NewProposal(t.old, initFact.Round(), nil, ivp) // new proposal from old
	t.NoError(st.ProcessProposal(pr))

	select {
	case <-time.After(time.Second * 2):
	case <-sealch:
		t.NoError(errors.Errorf("timeout to wait to broadcast new accept, but it should not"))
	}
}

func (t *testStateHandover) TestNewProposalNextAcceptBallotBroadcastedAfterJoined() {
	st, done := t.newState(t.Suffrage(t.local, t.remote), nil)
	defer done()

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	sealch := make(chan seal.Seal, 1)
	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		if _, ok := sl.(ballot.ACCEPT); !ok {
			return nil
		}

		sealch <- sl

		return nil
	})

	t.local.Policy().SetTimeoutWaitingProposal(time.Millisecond * 10)
	t.local.Policy().SetWaitBroadcastingACCEPTBallot(time.Nanosecond)
	f, err := st.Enter(NewStateSwitchContext(base.StateSyncing, base.StateHandover).
		SetVoteproof(ivp),
	)
	t.NoError(err)
	t.NoError(f())

	_ = st.jivp.Set(ivp)
	pr := t.NewProposal(t.old, initFact.Round(), nil, ivp) // new proposal from old
	t.NoError(st.ProcessProposal(pr))

	select {
	case <-time.After(time.Second * 2):
		t.NoError(errors.Errorf("timeout to wait to broadcast new accept"))
	case <-sealch:
	}
}

// TestNoNewProposal tests,
// - HandoverState also will wait proposal from other suffrage nodes, including
// old node
// - If no expected Proposal, HandoverState tries to move next round like
// ConsensusState
func (t *testStateHandover) TestNoNewProposal() {
	st, done := t.newState(t.Suffrage(t.local, t.remote), nil)
	defer done()

	sealch := make(chan seal.Seal, 1)
	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		if _, ok := sl.(ballot.INIT); !ok {
			return nil
		}

		sealch <- sl

		return nil
	})

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITFactV0

	vp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	t.local.Policy().SetTimeoutWaitingProposal(time.Millisecond * 10)
	f, err := st.Enter(NewStateSwitchContext(base.StateSyncing, base.StateHandover).
		SetVoteproof(vp))
	t.NoError(err)
	t.NoError(f())

	select {
	case <-time.After(time.Second * 2):
		t.NoError(errors.Errorf("timeout to wait next round INIT ballot"))
	case sl := <-sealch:
		t.Implements((*ballot.INIT)(nil), sl)

		bb, ok := sl.(ballot.INIT)
		t.True(ok)

		t.Equal(base.StageINIT, bb.Stage())

		t.Equal(vp.Height(), bb.Height())
		t.Equal(vp.Round()+1, bb.Round())

		previousManifest, found, err := t.local.Database().ManifestByHeight(vp.Height() - 1)
		t.True(found)
		t.NoError(err)
		t.NotNil(previousManifest)

		fact := bb.Fact().(ballot.INITFact)
		t.True(previousManifest.Hash().Equal(fact.PreviousBlock()))
	}
}

func (t *testStateHandover) TestMoveToConsensus() {
	st, done := t.newState(t.Suffrage(t.remote, t.local), nil) // NOTE set local is not proposer
	defer done()

	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		return nil
	})

	st.SetNewBlocksFunc(func(blks []block.Block) error {
		return nil
	})

	var ivp, avp base.Voteproof
	var pr ballot.Proposal

	{
		ivp = t.nextINITVoteproof(t.local, nil, nil, base.Round(0), t.local, t.remote)
		pr = t.NewProposal(t.local, ivp.Round(), nil, ivp)
		t.NoError(t.local.Database().NewProposal(pr))
	}

	f, err := st.Enter(NewStateSwitchContext(base.StateSyncing, base.StateHandover).
		SetVoteproof(ivp))
	t.NoError(err)
	t.NoError(f())

	t.NoError(st.ProcessProposal(pr))
	<-time.After(time.Second)
	avp = t.nextACCEPTVoteproof(t.local, pr, st.pps.Current().Block().Hash(), t.local, t.remote)

	t.NoError(st.ProcessVoteproof(avp))

	_ = st.States.hd.st.setUnderHandover(true) // NOTE under handover
	st.States.dis.SetJoined(true)              // NOTE join discovery
	_ = st.af.Set(true)

	ivp = t.nextINITVoteproof(t.local, ivp, avp, base.Round(0), t.local, t.remote)
	t.NoError(st.ProcessVoteproof(ivp))

	t.True(t.checkJoined(st)) // check joined

	t.NotNil(st.joinedINITVoteproof())

	pr = t.NewProposal(t.local, ivp.Round(), nil, ivp)
	t.NoError(t.local.Database().NewProposal(pr))
	t.NoError(st.ProcessProposal(pr))
	<-time.After(time.Second)
	avp = t.nextACCEPTVoteproof(t.local, pr, st.pps.Current().Block().Hash(), t.local, t.remote)
	t.NoError(st.ProcessVoteproof(avp))

	ivp = t.nextINITVoteproof(t.local, ivp, avp, base.Round(0), t.local, t.remote)
	err = st.ProcessVoteproof(ivp)

	var sctx StateSwitchContext
	t.True(errors.As(err, &sctx))
	t.Equal(base.StateConsensus, sctx.ToState())
	t.Equal(0, base.CompareVoteproof(ivp, sctx.Voteproof()))
}

func (t *testStateHandover) TestUpdatePassthroughFilter() {
	st, done := t.newState(t.Suffrage(t.remote, t.local), nil) // NOTE set local is not proposer
	defer done()

	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		return nil
	})

	st.SetNewBlocksFunc(func(blks []block.Block) error {
		return nil
	})

	var ivp, avp base.Voteproof
	var pr ballot.Proposal

	{
		ivp = t.nextINITVoteproof(t.local, nil, nil, base.Round(0), t.local, t.remote)
		pr = t.NewProposal(t.local, ivp.Round(), nil, ivp)
		t.NoError(t.local.Database().NewProposal(pr))
	}

	f, err := st.Enter(NewStateSwitchContext(base.StateSyncing, base.StateHandover).
		SetVoteproof(ivp))
	t.NoError(err)
	t.NoError(f())

	t.NoError(st.ProcessProposal(pr))
	<-time.After(time.Second)
	avp = t.nextACCEPTVoteproof(t.local, pr, st.pps.Current().Block().Hash(), t.local, t.remote)

	t.NoError(st.ProcessVoteproof(avp))

	st.States.dis.SetJoined(true) // NOTE join discovery
	st.States.hd.st.setOldNode(t.old.Channel())

	t.True(t.checkJoined(st))

	ivp = t.nextINITVoteproof(t.local, ivp, avp, base.Round(0), t.local, t.remote)
	t.NoError(st.ProcessVoteproof(ivp))
	t.NotNil(st.joinedINITVoteproof())

	t.Run("normal operation should be passed", func() {
		op, err := operation.NewKVOperation(t.local.Node().Privatekey(), nil, "a", nil, nil)
		t.NoError(err)
		sl, err := operation.NewBaseSeal(t.local.Node().Privatekey(), []operation.Operation{op}, nil)
		t.NoError(err)

		psl := network.NewPassthroughedSealFromConnInfo(sl, t.local.Channel().ConnInfo())

		oldreceivedch := make(chan seal.Seal, 1)
		t.NoError(st.nodepool.Passthroughs(context.Background(), psl, func(sl seal.Seal, ch network.Channel) {
			if !ch.ConnInfo().Equal(t.old.Channel().ConnInfo()) {
				return
			}

			oldreceivedch <- sl
		}))

		select {
		case <-time.After(time.Second * 2):
			t.NoError(errors.Errorf("timeout to wait to be passthroughed to old"))
		case rsl := <-oldreceivedch:
			t.True(sl.Hash().Equal(rsl.Hash()))
		}
	})

	t.Run("current INIT ballot should be passed", func() {
		ib := t.NewINITBallot(t.local, base.Round(0), nil)
		psl := network.NewPassthroughedSealFromConnInfo(ib, t.local.Channel().ConnInfo())

		oldreceivedch := make(chan seal.Seal, 1)
		t.NoError(st.nodepool.Passthroughs(context.Background(), psl, func(sl seal.Seal, ch network.Channel) {
			if !ch.ConnInfo().Equal(t.old.Channel().ConnInfo()) {
				return
			}

			oldreceivedch <- sl
		}))

		select {
		case <-time.After(time.Second * 2):
			t.NoError(errors.Errorf("timeout to wait to be passthroughed to old"))
		case rsl := <-oldreceivedch:
			t.True(ib.Hash().Equal(rsl.Hash()))
		}
	})

	t.Run("old height ballot should be filtered", func() {
		ib := ballot.NewINITV0(
			t.local.Node().Address(),
			ivp.Height()-1,
			base.Round(0),
			valuehash.RandomSHA256(),
			nil,
			nil,
		)
		t.NoError(ib.Sign(t.local.Node().Privatekey(), t.local.Policy().NetworkID()))

		psl := network.NewPassthroughedSealFromConnInfo(ib, t.local.Channel().ConnInfo())

		oldreceivedch := make(chan seal.Seal, 1)
		t.NoError(st.nodepool.Passthroughs(context.Background(), psl, func(sl seal.Seal, ch network.Channel) {
			if !ch.ConnInfo().Equal(t.old.Channel().ConnInfo()) {
				return
			}

			oldreceivedch <- sl
		}))

		select {
		case <-time.After(time.Second * 2):
		case <-oldreceivedch:
			t.NoError(errors.Errorf("old INIT ballot passthroughed to old"))
		}
	})

	t.Run("same height, but higher round INIT ballot should be filtered", func() {
		ib := ballot.NewINITV0(
			t.local.Node().Address(),
			ivp.Height(),
			ivp.Round()+1,
			valuehash.RandomSHA256(),
			nil,
			nil,
		)
		t.NoError(ib.Sign(t.local.Node().Privatekey(), t.local.Policy().NetworkID()))

		psl := network.NewPassthroughedSealFromConnInfo(ib, t.local.Channel().ConnInfo())

		oldreceivedch := make(chan seal.Seal, 1)
		t.NoError(st.nodepool.Passthroughs(context.Background(), psl, func(sl seal.Seal, ch network.Channel) {
			if !ch.ConnInfo().Equal(t.old.Channel().ConnInfo()) {
				return
			}

			oldreceivedch <- sl
		}))

		select {
		case <-time.After(time.Second * 2):
		case <-oldreceivedch:
			t.NoError(errors.Errorf("same height, but higher INIT ballot passthroughed to old"))
		}
	})

	t.Run("same height, but not ACCEPT ballot should be filtered", func() {
		ab := ballot.NewACCEPTV0(
			t.local.Node().Address(),
			ivp.Height(),
			ivp.Round(),
			valuehash.RandomSHA256(),
			valuehash.RandomSHA256(),
			ivp,
		)

		t.NoError(ab.Sign(t.local.Node().Privatekey(), t.local.Policy().NetworkID()))

		psl := network.NewPassthroughedSealFromConnInfo(ab, t.local.Channel().ConnInfo())

		oldreceivedch := make(chan seal.Seal, 1)
		t.NoError(st.nodepool.Passthroughs(context.Background(), psl, func(sl seal.Seal, ch network.Channel) {
			if !ch.ConnInfo().Equal(t.old.Channel().ConnInfo()) {
				return
			}

			oldreceivedch <- sl
		}))

		select {
		case <-time.After(time.Second * 2):
		case <-oldreceivedch:
			t.NoError(errors.Errorf("same height, but ACCEPT ballot passthroughed to old"))
		}
	})
}

func (t *testStateHandover) TestEmptyRemotes() {
	st, done := t.newState(t.Suffrage(t.local), nil) // NOTE set local is not proposer
	defer done()

	joinDiscoveryCalled := make(chan struct{}, 1)
	st.States.joinDiscoveryFunc = func(int, chan error) error {
		joinDiscoveryCalled <- struct{}{}

		return nil
	}

	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		return nil
	})

	st.SetNewBlocksFunc(func(blks []block.Block) error {
		return nil
	})

	var ivp base.Voteproof
	var pr ballot.Proposal

	{
		ivp = t.nextINITVoteproof(t.local, nil, nil, base.Round(0), t.local, t.remote)
		pr = t.NewProposal(t.local, ivp.Round(), nil, ivp)
		t.NoError(t.local.Database().NewProposal(pr))
	}

	f, err := st.Enter(NewStateSwitchContext(base.StateSyncing, base.StateHandover).
		SetVoteproof(ivp))
	t.NoError(err)
	t.NoError(f())

	select {
	case <-time.After(time.Second * 2):
	case <-joinDiscoveryCalled:
		t.NoError(errors.Errorf("when remotes empty, joinDiscovery should not be called"))
	}
}

func (t *testStateHandover) TestFinishEndHandoverSeal() {
	st, done := t.newState(t.Suffrage(t.remote, t.local), nil) // NOTE set local is not proposer
	defer done()

	oldch := t.old.Channel().(*channetwork.Channel)
	st.States.hd.st.setOldNode(oldch)

	sealch := make(chan network.EndHandoverSeal, 1)
	oldch.SetEndHandover(func(sl network.EndHandoverSeal) (bool, error) {
		sealch <- sl

		return true, nil
	})

	t.NoError(st.nodepool.SetPassthrough(oldch, func(network.PassthroughedSeal) bool {
		return true
	}, -1))

	sctx := NewStateSwitchContext(base.StateHandover, base.StateConsensus)
	f, err := st.Exit(sctx)
	t.NoError(err)
	t.NoError(f())

	select {
	case <-time.After(time.Second * 2):
		t.NoError(errors.Errorf("failed to wait EndHandoverSeal for old node"))
	case sl := <-sealch:
		t.NoError(sl.IsValid(t.local.Policy().NetworkID()))
		t.True(sl.Hint().Equal(network.EndHandoverSealV0Hint))
		t.True(sl.Signer().Equal(t.local.Node().Publickey()))
		t.True(sl.ConnInfo().Equal(t.local.Channel().ConnInfo()))
	}

	t.False(st.States.hd.IsStarted())
	t.False(st.States.underHandover())
	t.False(st.States.isHandoverReady())
	t.False(st.nodepool.ExistsPassthrough(oldch.ConnInfo()))
}

func TestStateHandover(t *testing.T) {
	suite.Run(t, new(testStateHandover))
}
