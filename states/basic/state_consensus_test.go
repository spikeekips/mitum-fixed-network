package basicstates

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/tree"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testStateConsensus struct {
	baseTestState
	local  *isaac.Local
	remote *isaac.Local
}

func (t *testStateConsensus) SetupTest() {
	ls := t.Locals(2)
	t.local, t.remote = ls[0], ls[1]
}

func (t *testStateConsensus) newState(suffrage base.Suffrage, pps *prprocessor.Processors) (*ConsensusState, func()) {
	if suffrage == nil {
		suffrage = t.Suffrage(t.remote, t.local)
	}

	if pps == nil {
		pps = t.DummyProcessors()
	}

	proposalMaker := isaac.NewProposalMaker(t.local.Node(), t.local.Storage(), t.local.Policy())
	st := NewConsensusState(
		t.local.Node(),
		t.local.Storage(),
		t.local.BlockFS(),
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

	lastINITVoteproof, found, err := t.local.BlockFS().LastVoteproof(base.StageINIT)
	t.NoError(err)
	t.True(found)

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
	initFact := ib.INITBallotFactV0

	vp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	_, err = st.Enter(NewStateSwitchContext(base.StateJoining, base.StateConsensus).SetVoteproof(vp))
	t.Contains(err.Error(), "empty last init voteproof")
}

func (t *testStateConsensus) TestEnterWithWrongVoteproof() {
	st, done := t.newState(nil, nil)
	defer done()

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	vp, err := t.NewVoteproof(base.StageACCEPT, initFact, t.local, t.remote)
	t.NoError(err)
	st.SetLastVoteproofFuncs(func() base.Voteproof { return vp }, func() base.Voteproof { return vp }, nil)

	lastAcceptVoteproof, found, err := t.local.BlockFS().LastVoteproof(base.StageACCEPT)
	t.NoError(err)
	t.True(found)

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
	initFact := ib.INITBallotFactV0

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
	initFact := ib.INITBallotFactV0

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

	sealch := make(chan seal.Seal)
	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		if _, ok := sl.(ballot.Proposal); !ok {
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
		t.NoError(xerrors.Errorf("timeout to wait proposal"))
	case sl := <-sealch:
		t.Implements((*ballot.Proposal)(nil), sl)

		bb, ok := sl.(ballot.Proposal)
		t.True(ok)

		t.Equal(vp.Height(), bb.Height())
		t.Equal(vp.Round(), bb.Round())
	}
}

// TestFindProposal tests,
// - state receives valid init voteproof
// - local is not proposer
// - if proper proposal in local, process it.
func (t *testStateConsensus) TestFindProposal() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	vp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, initFact.Round(), nil, vp)

	// NOTE save proposal in local
	t.NoError(t.local.Storage().NewProposal(pr))

	newblock, _ := block.NewTestBlockV0(vp.Height(), vp.Round(), pr.Hash(), valuehash.RandomSHA256())

	dp := &prprocessor.DummyProcessor{S: prprocessor.BeforePrepared}
	dp.PF = func(ctx context.Context) (block.Block, error) {
		return newblock, nil
	}
	pps := t.Processors(dp.New)

	st, done := t.newState(nil, pps) // NOTE set local is not proposer
	defer done()

	sealch := make(chan seal.Seal)
	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		if _, ok := sl.(ballot.ACCEPTBallot); !ok {
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
		t.NoError(xerrors.Errorf("timeout to wait accept ballot"))
	case sl := <-sealch:
		t.Implements((*ballot.ACCEPTBallot)(nil), sl)

		bb, ok := sl.(ballot.ACCEPTBallot)
		t.True(ok)

		t.Equal(base.StageACCEPT, bb.Stage())

		t.Equal(vp.Height(), bb.Height())
		t.Equal(vp.Round(), bb.Round())

		fact := bb.Fact().(ballot.ACCEPTBallotFact)
		t.True(newblock.Hash().Equal(fact.NewBlock()))
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

	sealch := make(chan seal.Seal)
	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		if _, ok := sl.(ballot.INITBallot); !ok {
			return nil
		}

		sealch <- sl

		return nil
	})

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	vp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	t.local.Policy().SetTimeoutWaitingProposal(time.Millisecond * 10)
	f, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateConsensus).SetVoteproof(vp))
	t.NoError(err)
	t.NoError(f())

	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("timeout to wait init ballot"))
	case sl := <-sealch:
		t.Implements((*ballot.INITBallot)(nil), sl)

		bb, ok := sl.(ballot.INITBallot)
		t.True(ok)

		t.Equal(base.StageINIT, bb.Stage())

		t.Equal(vp.Height(), bb.Height())
		t.Equal(vp.Round()+1, bb.Round())

		previousManifest, found, err := t.local.Storage().ManifestByHeight(vp.Height() - 1)
		t.True(found)
		t.NoError(err)
		t.NotNil(previousManifest)

		fact := bb.Fact().(ballot.INITBallotFact)
		t.True(previousManifest.Hash().Equal(fact.PreviousBlock()))
	}
}

// TestStuckACCEPTVotingKeepBroadcastingACCEPTAndProposal test,
// - stuck at accept voting
// - state keeps waiting new accept ballot
// - state keeps broadcasting accept ballot and proposal
func (t *testStateConsensus) TestStuckACCEPTVotingKeepBroadcastingACCEPTAndProposal() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

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

	sealch := make(chan seal.Seal)
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
				if _, ok := sl.(ballot.ACCEPTBallot); !ok {
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
		case ballot.ACCEPTBallot, ballot.Proposal:
		default:
			t.NoError(xerrors.Errorf("unexpected ballot found, %T", sl))
		}
	}
}

func (t *testStateConsensus) TestACCEPTVoteproof() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, initFact.Round(), nil, ivp)

	// NOTE save proposal in local
	t.NoError(t.local.Storage().NewProposal(pr))

	newblock, _ := block.NewTestBlockV0(ivp.Height(), ivp.Round(), pr.Hash(), valuehash.RandomSHA256())

	var avp base.Voteproof
	{
		ab := t.NewACCEPTBallot(t.local, ivp.Round(), newblock.Proposal(), newblock.Hash(), nil)
		fact := ab.ACCEPTBallotFactV0

		avp, _ = t.NewVoteproof(base.StageACCEPT, fact, t.local, t.remote)
	}

	dp := &prprocessor.DummyProcessor{P: pr, S: prprocessor.BeforePrepared}
	dp.PF = func(ctx context.Context) (block.Block, error) {
		return newblock, nil
	}
	dp.SF = func(ctx context.Context) error {
		dp.B = newblock

		bs := storage.NewDummyBlockStorage(newblock, tree.FixedTree{}, tree.FixedTree{})

		return bs.Commit(ctx)
	}
	pps := t.Processors(dp.New)

	st, done := t.newState(nil, pps) // NOTE set local is proposer
	defer done()

	sealch := make(chan seal.Seal)
	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		sealch <- sl

		return nil
	})

	blockch := make(chan block.Block, 1)
	st.SetNewBlocksFunc(func(blks []block.Block) error {
		if len(blks) < 1 {
			return xerrors.Errorf("empty blocks")
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
			t.NoError(xerrors.Errorf("timeout to wait init ballot"))

			break end
		case sl := <-sealch:
			if _, ok := sl.(ballot.Proposal); ok {
				break end
			}
		}
	}

	t.NoError(st.ProcessVoteproof(avp))

	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("timeout to wait saved block"))
	case savedBlock := <-blockch:
		t.Equal(newblock.Height(), savedBlock.Height())
		t.True(newblock.Hash().Equal(savedBlock.Hash()))
	}
}

func (t *testStateConsensus) TestDrawACCEPTVoteproofToNextRound() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, initFact.Round(), nil, ivp)

	// NOTE save proposal in local
	t.NoError(t.local.Storage().NewProposal(pr))

	newblock, _ := block.NewTestBlockV0(ivp.Height(), ivp.Round(), pr.Hash(), valuehash.RandomSHA256())

	dp := &prprocessor.DummyProcessor{S: prprocessor.BeforePrepared}
	dp.PF = func(ctx context.Context) (block.Block, error) {
		return newblock, nil
	}
	dp.SF = func(ctx context.Context) error {
		dp.B = newblock

		bs := storage.NewDummyBlockStorage(newblock, tree.FixedTree{}, tree.FixedTree{})

		return bs.Commit(ctx)
	}
	pps := t.Processors(dp.New)

	st, done := t.newState(nil, pps)
	defer done()

	sealch := make(chan seal.Seal)
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
			t.NoError(xerrors.Errorf("timeout to wait init ballot"))

			break end
		case sl := <-sealch:
			if _, ok := sl.(ballot.Proposal); ok {
				break end
			}
		}
	}

	var drew base.Voteproof
	{
		dummyBlock, _ := block.NewTestBlockV0(ivp.Height(), ivp.Round(), valuehash.RandomSHA256(), valuehash.RandomSHA256())

		ab := t.NewACCEPTBallot(t.local, ivp.Round(), dummyBlock.Proposal(), dummyBlock.Hash(), nil)
		fact := ab.ACCEPTBallotFactV0

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
			if _, ok := sl.(ballot.INITBallot); !ok {
				continue
			}

			seals = append(seals, sl)
		}
	}
	t.NotEmpty(seals)
	sl := seals[len(seals)-1]

	t.Implements((*ballot.INITBallot)(nil), sl)

	bb, ok := sl.(ballot.INITBallot)
	t.True(ok)

	t.Equal(base.StageINIT, bb.Stage())

	t.Equal(ivp.Height(), bb.Height())
	t.Equal(ivp.Round()+1, bb.Round())
}

func (t *testStateConsensus) TestFailedSavingBlockMovesToSyncing() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, initFact.Round(), nil, ivp)

	// NOTE save proposal in local
	t.NoError(t.local.Storage().NewProposal(pr))

	newblock, _ := block.NewTestBlockV0(ivp.Height(), ivp.Round(), pr.Hash(), valuehash.RandomSHA256())

	var avp base.Voteproof
	{
		ab := t.NewACCEPTBallot(t.local, ivp.Round(), newblock.Proposal(), newblock.Hash(), nil)
		fact := ab.ACCEPTBallotFactV0

		avp, _ = t.NewVoteproof(base.StageACCEPT, fact, t.local, t.remote)
	}

	dp := &prprocessor.DummyProcessor{S: prprocessor.BeforePrepared}
	dp.PF = func(ctx context.Context) (block.Block, error) {
		return newblock, nil
	}
	dp.SF = func(ctx context.Context) error {
		return xerrors.Errorf("killme")
	}
	pps := t.Processors(dp.New)

	st, done := t.newState(nil, pps)
	defer done()

	sealch := make(chan seal.Seal)
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
			t.NoError(xerrors.Errorf("timeout to wait init ballot"))

			break end
		case sl := <-sealch:
			if _, ok := sl.(ballot.Proposal); ok {
				break end
			}
		}
	}

	// NOTE insert draw accept voteproof
	err = st.ProcessVoteproof(avp)

	var sctx StateSwitchContext
	t.True(xerrors.As(err, &sctx))
	t.Equal(base.StateSyncing, sctx.ToState())
}

func (t *testStateConsensus) TestFailedProcessingProposal() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, initFact.Round(), nil, ivp)

	t.NoError(t.local.Storage().NewProposal(pr))

	dp := &prprocessor.DummyProcessor{S: prprocessor.BeforePrepared}
	dp.PF = func(ctx context.Context) (block.Block, error) {
		return nil, xerrors.Errorf("happy meal")
	}
	pps := t.Processors(dp.New)

	st, done := t.newState(nil, pps)
	defer done()

	sealch := make(chan seal.Seal)
	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		sealch <- sl

		return nil
	})

	f, err := st.Enter(NewStateSwitchContext(base.StateJoining, base.StateConsensus).SetVoteproof(ivp))
	t.NoError(err)
	t.NoError(f())

	// NOTE proposal will be broadcasted prior to accept ballot
	var nib ballot.ACCEPTBallot

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

end:
	for {
		select {
		case <-ctx.Done():
			t.NoError(xerrors.Errorf("timeout to wait accept ballot"))

			break end
		case sl := <-sealch:
			if i, ok := sl.(ballot.ACCEPTBallot); !ok {
				continue
			} else if i.Height() == pr.Height() && i.Round() == pr.Round() {
				nib = i

				break end
			}
		}
	}

	t.Equal(pr.Height(), nib.Height())
	t.Equal(pr.Round(), nib.Round())

	t.True(bytes.HasSuffix(nib.Fact().(ballot.ACCEPTBallotFact).NewBlock().Bytes(), BlockPrefixFailedProcessProposal))
}

func (t *testStateConsensus) TestProcessingProposalFromACCEPTVoterpof() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, initFact.Round(), nil, ivp)

	// NOTE save proposal in local
	t.NoError(t.local.Storage().NewProposal(pr))

	newblock, _ := block.NewTestBlockV0(ivp.Height(), ivp.Round(), pr.Hash(), valuehash.RandomSHA256())

	var avp base.Voteproof
	{
		ab := t.NewACCEPTBallot(t.local, ivp.Round(), newblock.Proposal(), newblock.Hash(), nil)
		fact := ab.ACCEPTBallotFactV0

		avp, _ = t.NewVoteproof(base.StageACCEPT, fact, t.local, t.remote)
	}

	dp := &prprocessor.DummyProcessor{S: prprocessor.BeforePrepared}
	dp.PF = func(ctx context.Context) (block.Block, error) {
		return newblock, nil
	}
	dp.SF = func(ctx context.Context) error {
		dp.B = newblock

		bs := storage.NewDummyBlockStorage(newblock, tree.FixedTree{}, tree.FixedTree{})

		return bs.Commit(ctx)
	}
	pps := t.Processors(dp.New)

	st, done := t.newState(nil, pps) // NOTE set local is proposer
	defer done()

	sealch := make(chan seal.Seal)
	st.SetBroadcastSealsFunc(func(sl seal.Seal, toLocal bool) error {
		sealch <- sl

		return nil
	})

	blockch := make(chan block.Block, 1)
	st.SetNewBlocksFunc(func(blks []block.Block) error {
		if len(blks) < 1 {
			return xerrors.Errorf("empty blocks")
		}

		blockch <- blks[0]

		return nil
	})

	oldINITVoteproof, err := t.local.BlockFS().LoadINITVoteproof(st.LastINITVoteproof().Height() - 1)
	t.NoError(err)

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
			t.NoError(xerrors.Errorf("timeout to wait init ballot"))

			break end
		case sl := <-sealch:
			if _, ok := sl.(ballot.INITBallot); ok {
				break end
			}
		}
	}

	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("timeout to wait saved block"))
	case savedBlock := <-blockch:
		t.Equal(newblock.Height(), savedBlock.Height())
		t.True(newblock.Hash().Equal(savedBlock.Hash()))
	}
}

func TestStateConsensus(t *testing.T) {
	suite.Run(t, new(testStateConsensus))
}
