package basicstates

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/tree"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testStateConsensus struct {
	baseTestState
}

func (t *testStateConsensus) newState(suffrage base.Suffrage, pps *prprocessor.Processors) (*ConsensusState, func()) {
	if suffrage == nil {
		suffrage = t.Suffrage(t.remote, t.local)
	}

	if pps == nil {
		pps = t.DummyProcessors()
	}

	proposalMaker := isaac.NewProposalMaker(t.local.Node(), t.local.Database(), t.local.Policy())
	st := NewConsensusState(
		t.local.Database(),
		t.local.Policy(),
		t.local.Nodes(),
		suffrage,
		proposalMaker,
		pps,
	)

	timers := localtime.NewTimers([]localtime.TimerID{
		TimerIDBroadcastINITBallot,
		TimerIDBroadcastProposal,
		TimerIDBroadcastACCEPTBallot,
		TimerIDFindProposal,
	}, false)
	st.SetTimers(timers)

	lastINITVoteproof := t.local.Database().LastVoteproof(base.StageINIT)
	t.NotNil(lastINITVoteproof)

	st.SetLastVoteproofFuncs(func() base.Voteproof {
		return lastINITVoteproof
	}, func() base.Voteproof {
		return lastINITVoteproof
	}, nil)

	return st, func() {
		f, err := st.Exit(NewStateSwitchContext(base.StateConsensus, base.StateStopped))
		t.NoError(err)
		_ = f()

		_ = timers.Stop()
	}
}

func (t *testStateConsensus) TestEnterWithNilVoteproof() {
	st, done := t.newState(nil, nil)
	defer done()

	_, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateConsensus).SetVoteproof(nil))
	t.Contains(err.Error(), "enter without voteproof")
}

func (t *testStateConsensus) TestEnterAndNilLastINITVoteproof() {
	st, done := t.newState(nil, nil)
	defer done()

	st.SetLastVoteproofFuncs(func() base.Voteproof { return nil }, func() base.Voteproof { return nil }, nil)

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	vp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	_, err = st.Enter(NewStateSwitchContext(base.StateJoining, base.StateConsensus).SetVoteproof(vp))
	t.Contains(err.Error(), "empty last init voteproof")
}

func (t *testStateConsensus) TestEnterWithWrongVoteproof() {
	st, done := t.newState(nil, nil)
	defer done()

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	vp, err := t.NewVoteproof(base.StageACCEPT, initFact, t.local, t.remote)
	t.NoError(err)
	st.SetLastVoteproofFuncs(func() base.Voteproof { return vp }, func() base.Voteproof { return vp }, nil)

	lastAcceptVoteproof := t.local.Database().LastVoteproof(base.StageACCEPT)
	t.NotNil(lastAcceptVoteproof)

	_, err = st.Enter(NewStateSwitchContext(base.StateJoining, base.StateConsensus).SetVoteproof(lastAcceptVoteproof))
	t.Contains(err.Error(), "not allowed")
}

func (t *testStateConsensus) TestExit() {
	st, done := t.newState(nil, nil)
	defer done()

	st.SetBroadcastSealsFunc(func(seal.Seal, bool) error {
		return nil
	})

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	vp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	f, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateConsensus).SetVoteproof(vp))
	t.NoError(err)
	t.NoError(f())

	f, err = st.Exit(NewStateSwitchContext(base.StateConsensus, base.StateStopped))
	t.NoError(err)
	t.NoError(f())
	t.Empty(st.Timers().Started())
}

// TestBroadcastProposalWithINITVoteproof tests,
// - state receives valid init voteproof
// - local is proposer
// - broadcasts proposal
func (t *testStateConsensus) TestBroadcastProposalWithINITVoteproof() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	vp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	newblock, _ := block.NewTestBlockV0(vp.Height(), vp.Round(), valuehash.RandomSHA256(), valuehash.RandomSHA256())

	dp := &prprocessor.DummyProcessor{S: prprocessor.BeforePrepared}
	dp.PF = func(ctx context.Context) (block.Block, error) {
		return newblock, nil
	}
	pps := t.Processors(dp.New)

	st, done := t.newState(t.Suffrage(t.local, t.remote), pps) // NOTE set local is proposer
	defer done()

	sealch := make(chan seal.Seal, 1)
	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		if _, ok := sl.(base.Proposal); !ok {
			return nil
		}

		sealch <- sl

		return nil
	})

	f, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateConsensus).SetVoteproof(vp))
	t.NoError(err)
	t.NoError(f())

	// NOTE proposal will be broadcasted prior to accept ballot
	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("timeout to wait proposal"))
	case sl := <-sealch:
		t.Implements((*base.Proposal)(nil), sl)

		bb, ok := sl.(base.Proposal)
		t.True(ok)

		t.Equal(vp.Height(), bb.Fact().Height())
		t.Equal(vp.Round(), bb.Fact().Round())
	}
}

// TestFindProposal tests,
// - state receives valid init voteproof
// - local is not proposer
// - if proper proposal in local, process it.
func (t *testStateConsensus) TestFindProposal() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	vp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, initFact.Round(), nil, vp)

	// NOTE save proposal in local
	t.NoError(t.local.Database().NewProposal(pr))

	newblock, _ := block.NewTestBlockV0(vp.Height(), vp.Round(), pr.Fact().Hash(), valuehash.RandomSHA256())

	dp := &prprocessor.DummyProcessor{S: prprocessor.BeforePrepared}
	dp.PF = func(ctx context.Context) (block.Block, error) {
		return newblock, nil
	}
	pps := t.Processors(dp.New)

	st, done := t.newState(nil, pps) // NOTE set local is not proposer
	defer done()

	sealch := make(chan seal.Seal, 1)
	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		if _, ok := sl.(base.ACCEPTBallot); !ok {
			return nil
		}

		sealch <- sl

		return nil
	})

	f, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateConsensus).SetVoteproof(vp))
	t.NoError(err)
	t.NoError(f())

	// NOTE proposal will be broadcasted prior to accept ballot
	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("timeout to wait accept ballot"))
	case sl := <-sealch:
		t.Implements((*base.ACCEPTBallot)(nil), sl)

		bb, ok := sl.(base.ACCEPTBallot)
		t.True(ok)

		t.Equal(base.StageACCEPT, bb.Fact().Stage())

		t.Equal(vp.Height(), bb.Fact().Height())
		t.Equal(vp.Round(), bb.Fact().Round())

		t.True(newblock.Hash().Equal(bb.Fact().NewBlock()))
	}
}

// TestTimeoutWaitingProposal tests,
// - ConsensusState receives init voteproof and wait proposal
// - local is not proposer
// - timed out to wait proposal,
// - ConsensusState will try to start next round
func (t *testStateConsensus) TestTimeoutWaitingProposal() {
	st, done := t.newState(t.Suffrage(t.remote, t.local), nil) // NOTE set local is not proposer
	defer done()

	sealch := make(chan seal.Seal, 1)
	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		if _, ok := sl.(base.INITBallot); !ok {
			return nil
		}

		sealch <- sl

		return nil
	})

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	vp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	t.local.Policy().SetTimeoutWaitingProposal(time.Millisecond * 10)
	f, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateConsensus).SetVoteproof(vp))
	t.NoError(err)
	t.NoError(f())

	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("timeout to wait init ballot"))
	case sl := <-sealch:
		t.Implements((*base.INITBallot)(nil), sl)

		bb, ok := sl.(base.INITBallot)
		t.True(ok)

		t.Equal(base.StageINIT, bb.Fact().Stage())

		t.Equal(vp.Height(), bb.Fact().Height())
		t.Equal(vp.Round()+1, bb.Fact().Round())

		previousManifest, found, err := t.local.Database().ManifestByHeight(vp.Height() - 1)
		t.True(found)
		t.NoError(err)
		t.NotNil(previousManifest)

		t.True(previousManifest.Hash().Equal(bb.Fact().PreviousBlock()))
	}
}

// TestStuckACCEPTVotingKeepBroadcastingACCEPTAndProposal test,
// - stuck at accept voting
// - state keeps waiting new accept ballot
// - state keeps broadcasting accept ballot and proposal
func (t *testStateConsensus) TestStuckACCEPTVotingKeepBroadcastingACCEPTAndProposal() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	vp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	newblock, _ := block.NewTestBlockV0(vp.Height(), vp.Round(), valuehash.RandomSHA256(), valuehash.RandomSHA256())

	dp := &prprocessor.DummyProcessor{S: prprocessor.BeforePrepared}
	dp.PF = func(ctx context.Context) (block.Block, error) {
		return newblock, nil
	}
	pps := t.Processors(dp.New)

	st, done := t.newState(t.Suffrage(t.local, t.remote), pps) // NOTE set local is proposer
	defer done()

	sealch := make(chan seal.Seal, 1)
	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		sealch <- sl

		return nil
	})

	f, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateConsensus).SetVoteproof(vp))
	t.NoError(err)
	t.NoError(f())

	// NOTE proposal will be broadcasted prior to accept ballot
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var seals []seal.Seal
	var foundACCEPT bool
end:
	for {
		select {
		case <-ctx.Done():
			break end
		case sl := <-sealch:
			if !foundACCEPT {
				if _, ok := sl.(base.ACCEPTBallot); !ok {
					continue
				}

				foundACCEPT = true
			}

			seals = append(seals, sl)
		}
	}

	t.NotEmpty(seals)

	for i := range seals {
		sl := seals[i]
		switch sl.(type) {
		case base.ACCEPTBallot, base.Proposal:
		default:
			t.NoError(errors.Errorf("unexpected ballot found, %T", sl))
		}
	}
}

func (t *testStateConsensus) TestACCEPTVoteproof() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, initFact.Round(), nil, ivp)

	// NOTE save proposal in local
	t.NoError(t.local.Database().NewProposal(pr))

	newblock, _ := block.NewTestBlockV0(ivp.Height(), ivp.Round(), pr.Fact().Hash(), valuehash.RandomSHA256())

	var avp base.Voteproof
	{
		ab := t.NewACCEPTBallot(t.local, ivp.Round(), pr.Fact().Hash(), newblock.Hash(), nil)
		fact := ab.Fact()

		avp, _ = t.NewVoteproof(base.StageACCEPT, fact, t.local, t.remote)
	}

	dp := &prprocessor.DummyProcessor{P: pr.SignedFact(), S: prprocessor.BeforePrepared}
	dp.PF = func(ctx context.Context) (block.Block, error) {
		return newblock, nil
	}
	dp.SF = func(ctx context.Context) error {
		dp.B = newblock

		bs := storage.NewDummyDatabaseSession(newblock, tree.EmptyFixedTree(), tree.EmptyFixedTree())

		return bs.Commit(ctx)
	}
	pps := t.Processors(dp.New)

	st, done := t.newState(nil, pps) // NOTE set local is proposer
	defer done()

	sealch := make(chan seal.Seal, 1)
	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		sealch <- sl

		return nil
	})

	blockch := make(chan block.Block, 1)
	st.SetNewBlocksFunc(func(blks []block.Block) error {
		if len(blks) < 1 {
			return errors.Errorf("empty blocks")
		}

		blockch <- blks[0]

		return nil
	})

	f, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateConsensus).SetVoteproof(ivp))
	t.NoError(err)
	t.NoError(f())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	// NOTE proposal will be broadcasted prior to accept ballot
end:
	for {
		select {
		case <-ctx.Done():
			t.NoError(errors.Errorf("timeout to wait init ballot"))

			break end
		case sl := <-sealch:
			if _, ok := sl.(base.Proposal); ok {
				break end
			}
		}
	}

	t.NoError(st.ProcessVoteproof(avp))

	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("timeout to wait saved block"))
	case savedBlock := <-blockch:
		t.Equal(newblock.Height(), savedBlock.Height())
		t.True(newblock.Hash().Equal(savedBlock.Hash()))
	}
}

func (t *testStateConsensus) TestDrawACCEPTVoteproofToNextRound() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, initFact.Round(), nil, ivp)

	// NOTE save proposal in local
	t.NoError(t.local.Database().NewProposal(pr))

	newblock, _ := block.NewTestBlockV0(ivp.Height(), ivp.Round(), pr.Fact().Hash(), valuehash.RandomSHA256())

	dp := &prprocessor.DummyProcessor{S: prprocessor.BeforePrepared}
	dp.PF = func(ctx context.Context) (block.Block, error) {
		return newblock, nil
	}
	dp.SF = func(ctx context.Context) error {
		dp.B = newblock

		bs := storage.NewDummyDatabaseSession(newblock, tree.EmptyFixedTree(), tree.EmptyFixedTree())

		return bs.Commit(ctx)
	}
	pps := t.Processors(dp.New)

	st, done := t.newState(nil, pps)
	defer done()

	sealch := make(chan seal.Seal, 1)
	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		sealch <- sl

		return nil
	})

	f, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateConsensus).SetVoteproof(ivp))
	t.NoError(err)
	t.NoError(f())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	// NOTE proposal will be broadcasted prior to accept ballot
end:
	for {
		select {
		case <-ctx.Done():
			t.NoError(errors.Errorf("timeout to wait init ballot"))

			break end
		case sl := <-sealch:
			if _, ok := sl.(base.Proposal); ok {
				break end
			}
		}
	}

	var drew base.Voteproof
	{
		dummyBlock, _ := block.NewTestBlockV0(ivp.Height(), ivp.Round(), valuehash.RandomSHA256(), valuehash.RandomSHA256())

		ab := t.NewACCEPTBallot(t.local, ivp.Round(), valuehash.RandomSHA256(), dummyBlock.Hash(), nil)
		fact := ab.Fact()

		i, _ := t.NewVoteproof(base.StageINIT, fact, t.local, t.remote)
		i.SetResult(base.VoteResultDraw)

		drew = i
	}

	// NOTE insert draw accept voteproof
	t.NoError(st.ProcessVoteproof(drew))

	ctx, cancel = context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	var seals []seal.Seal
end0:
	for {
		select {
		case <-ctx.Done():
			break end0
		case sl := <-sealch:
			if _, ok := sl.(base.INITBallot); !ok {
				continue
			}

			seals = append(seals, sl)
		}
	}
	t.NotEmpty(seals)
	sl := seals[len(seals)-1]

	t.Implements((*base.INITBallot)(nil), sl)

	bb, ok := sl.(base.INITBallot)
	t.True(ok)

	t.Equal(base.StageINIT, bb.Fact().Stage())

	t.Equal(ivp.Height(), bb.Fact().Height())
	t.Equal(ivp.Round()+1, bb.Fact().Round())
}

func (t *testStateConsensus) TestFailedSavingBlockMovesToSyncing() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, initFact.Round(), nil, ivp)

	// NOTE save proposal in local
	t.NoError(t.local.Database().NewProposal(pr))

	newblock, _ := block.NewTestBlockV0(ivp.Height(), ivp.Round(), pr.Fact().Hash(), valuehash.RandomSHA256())

	var avp base.Voteproof
	{
		ab := t.NewACCEPTBallot(t.local, ivp.Round(), pr.Fact().Hash(), newblock.Hash(), nil)
		fact := ab.Fact()

		avp, _ = t.NewVoteproof(base.StageACCEPT, fact, t.local, t.remote)
	}

	dp := &prprocessor.DummyProcessor{S: prprocessor.BeforePrepared}
	dp.PF = func(ctx context.Context) (block.Block, error) {
		return newblock, nil
	}
	dp.SF = func(ctx context.Context) error {
		return errors.Errorf("killme")
	}
	pps := t.Processors(dp.New)

	st, done := t.newState(nil, pps)
	defer done()

	sealch := make(chan seal.Seal, 1)
	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		sealch <- sl

		return nil
	})

	f, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateConsensus).SetVoteproof(ivp))
	t.NoError(err)
	t.NoError(f())

	// NOTE proposal will be broadcasted prior to accept ballot
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

end:
	for {
		select {
		case <-ctx.Done():
			t.NoError(errors.Errorf("timeout to wait init ballot"))

			break end
		case sl := <-sealch:
			if _, ok := sl.(base.Proposal); ok {
				break end
			}
		}
	}

	// NOTE insert draw accept voteproof
	err = st.ProcessVoteproof(avp)

	var sctx StateSwitchContext
	t.True(errors.As(err, &sctx))
	t.Equal(base.StateSyncing, sctx.ToState())
}

func (t *testStateConsensus) TestFailedProcessingProposal() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, initFact.Round(), nil, ivp)

	t.NoError(t.local.Database().NewProposal(pr))

	dp := &prprocessor.DummyProcessor{S: prprocessor.BeforePrepared}
	dp.PF = func(ctx context.Context) (block.Block, error) {
		return nil, errors.Errorf("happy meal")
	}
	pps := t.Processors(dp.New)

	st, done := t.newState(nil, pps)
	defer done()

	sealch := make(chan seal.Seal, 1)
	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		sealch <- sl

		return nil
	})

	f, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateConsensus).SetVoteproof(ivp))
	t.NoError(err)
	t.NoError(f())

	// NOTE proposal will be broadcasted prior to accept ballot
	var nib base.ACCEPTBallot

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

end:
	for {
		select {
		case <-ctx.Done():
			t.NoError(errors.Errorf("timeout to wait accept ballot"))

			break end
		case sl := <-sealch:
			if i, ok := sl.(base.ACCEPTBallot); !ok {
				continue
			} else if i.Fact().Height() == pr.Fact().Height() && i.Fact().Round() == pr.Fact().Round() {
				nib = i

				break end
			}
		}
	}

	t.Equal(pr.Fact().Height(), nib.Fact().Height())
	t.Equal(pr.Fact().Round(), nib.Fact().Round())

	t.True(bytes.HasSuffix(nib.Fact().NewBlock().Bytes(), BlockPrefixFailedProcessProposal))
}

func (t *testStateConsensus) TestProcessingProposalFromACCEPTVoterpof() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, initFact.Round(), nil, ivp)

	// NOTE save proposal in local
	t.NoError(t.local.Database().NewProposal(pr))

	newblock, _ := block.NewTestBlockV0(ivp.Height(), ivp.Round(), pr.Fact().Hash(), valuehash.RandomSHA256())

	var avp base.Voteproof
	{
		ab := t.NewACCEPTBallot(t.local, ivp.Round(), pr.Fact().Hash(), newblock.Hash(), nil)
		fact := ab.Fact()

		avp, _ = t.NewVoteproof(base.StageACCEPT, fact, t.local, t.remote)
	}

	dp := &prprocessor.DummyProcessor{S: prprocessor.BeforePrepared}
	dp.PF = func(ctx context.Context) (block.Block, error) {
		return newblock, nil
	}
	dp.SF = func(ctx context.Context) error {
		dp.B = newblock

		bs := storage.NewDummyDatabaseSession(newblock, tree.EmptyFixedTree(), tree.EmptyFixedTree())

		return bs.Commit(ctx)
	}
	pps := t.Processors(dp.New)

	st, done := t.newState(nil, pps) // NOTE set local is proposer
	defer done()

	sealch := make(chan seal.Seal, 1)
	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		sealch <- sl

		return nil
	})

	blockch := make(chan block.Block, 1)
	st.SetNewBlocksFunc(func(blks []block.Block) error {
		if len(blks) < 1 {
			return errors.Errorf("empty blocks")
		}

		blockch <- blks[0]

		return nil
	})

	oldINITVoteproof, err := t.local.Database().Voteproof(st.LastINITVoteproof().Height()-1, base.StageINIT)
	t.NoError(err)
	t.NotNil(oldINITVoteproof)

	_, err = st.Enter(NewStateSwitchContext(base.StateJoining, base.StateConsensus).SetVoteproof(oldINITVoteproof))
	t.NoError(err)

	t.NoError(st.ProcessVoteproof(avp))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	// NOTE proposal will be broadcasted prior to accept ballot
end:
	for {
		select {
		case <-ctx.Done():
			t.NoError(errors.Errorf("timeout to wait init ballot"))

			break end
		case sl := <-sealch:
			if _, ok := sl.(base.INITBallot); ok {
				break end
			}
		}
	}

	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("timeout to wait saved block"))
	case savedBlock := <-blockch:
		t.Equal(newblock.Height(), savedBlock.Height())
		t.True(newblock.Hash().Equal(savedBlock.Hash()))
	}
}

func (t *testStateConsensus) TestEnterUnderHandover() {
	suffrage := t.Suffrage(t.local, t.remote)

	st, done := t.newState(suffrage, nil)
	defer done()

	stt := t.newStates(t.local, suffrage, st)
	st.States = stt
	hd := NewHandover(t.local.Channel().ConnInfo(), t.Encs, t.local.Policy(), t.local.Nodes(), suffrage)
	_ = hd.st.setUnderHandover(true) // NOTE underhandover

	st.States.hd = hd

	_, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateConsensus))
	t.Contains(err.Error(), "consensus should not be entered under handover")
}

func (t *testStateConsensus) TestBroadcastProposalWithINITVoteproofNotUnderhandover() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	vp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	newblock, _ := block.NewTestBlockV0(vp.Height(), vp.Round(), valuehash.RandomSHA256(), valuehash.RandomSHA256())

	dp := &prprocessor.DummyProcessor{S: prprocessor.BeforePrepared}
	dp.PF = func(ctx context.Context) (block.Block, error) {
		return newblock, nil
	}
	pps := t.Processors(dp.New)

	suffrage := t.Suffrage(t.local, t.remote)
	st, done := t.newState(suffrage, pps) // NOTE set local is proposer
	defer done()

	stt := t.newStates(t.local, suffrage, st)
	st.States = stt
	hd := NewHandover(t.local.Channel().ConnInfo(), t.Encs, t.local.Policy(), t.local.Nodes(), suffrage)

	st.States.hd = hd

	sealch := make(chan seal.Seal, 1)
	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		if _, ok := sl.(base.Proposal); !ok {
			return nil
		}

		sealch <- sl

		return nil
	})

	st.BaseState.enterFunc = func(StateSwitchContext) (func() error, error) {
		_ = hd.st.setUnderHandover(true) // NOTE underhandover

		return nil, nil
	}

	f, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateConsensus).SetVoteproof(vp))
	t.NoError(err)
	t.NoError(f())

	// NOTE proposal will not be broadcasted prior to accept ballot
	select {
	case <-time.After(time.Second * 3):
	case <-sealch:
		t.NoError(errors.Errorf("proposal should be blocked"))
	}
}

func TestStateConsensus(t *testing.T) {
	suite.Run(t, new(testStateConsensus))
}
