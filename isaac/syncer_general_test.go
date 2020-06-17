package isaac

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/network"
	channetwork "github.com/spikeekips/mitum/network/gochan"
)

type testGeneralSyncer struct {
	baseTestSyncer
}

func (t *testGeneralSyncer) TestInvalidFrom() {
	base := t.lastManifest(t.localstate.Storage()).Height()
	{ // lower than base
		_, err := NewGeneralSyncer(t.localstate, []network.Node{t.remoteState.Node()}, base-1, base+2)
		t.Contains(err.Error(), "lower than last block")
	}

	{ // same with base
		_, err := NewGeneralSyncer(t.localstate, []network.Node{t.remoteState.Node()}, base, base+2)
		t.Contains(err.Error(), "same or lower than last block")
	}

	{ // higher than to
		_, err := NewGeneralSyncer(t.localstate, []network.Node{t.remoteState.Node()}, base+3, base+2)
		t.Contains(err.Error(), "higher than to height")
	}
}

func (t *testGeneralSyncer) TestInvalidSourceNodes() {
	ls := t.localstates(2)
	localstate, _ := ls[0], ls[1]

	t.setup(localstate, nil)

	base := t.lastManifest(localstate.Storage()).Height()

	{ // nil node
		_, err := NewGeneralSyncer(localstate, nil, base+1, base+2)
		t.Contains(err.Error(), "empty source nodes")
	}

	{ // same with local node
		_, err := NewGeneralSyncer(localstate, []network.Node{localstate.Node()}, base+1, base+2)
		t.Contains(err.Error(), "same with local node")
	}
}

func (t *testGeneralSyncer) TestNew() {
	ls := t.localstates(2)
	localstate, remoteState := ls[0], ls[1]

	t.setup(localstate, []*Localstate{remoteState})

	target := t.lastManifest(localstate.Storage()).Height() + 1
	t.generateBlocks([]*Localstate{remoteState}, target)

	cs, err := NewGeneralSyncer(localstate, []network.Node{remoteState.Node()}, target, target)
	t.NoError(err)

	defer cs.Close()

	_ = (interface{})(cs).(Syncer)
	t.Implements((*Syncer)(nil), cs)

	t.Equal(SyncerCreated, cs.State())
}

// TestHeadAndTailManifests setups 4 nodes and 3 nodes has higher blocks rather
// than 1 node.
func (t *testGeneralSyncer) TestHeadAndTailManifests() {
	ls := t.localstates(4)
	localstate, rn0, rn1, rn2 := ls[0], ls[1], ls[2], ls[3]

	t.setup(localstate, []*Localstate{rn0, rn1, rn2})

	base := t.lastManifest(localstate.Storage()).Height()
	target := base + 5
	t.generateBlocks([]*Localstate{rn0, rn1, rn2}, target)

	cs, err := NewGeneralSyncer(localstate, []network.Node{rn0.Node(), rn1.Node(), rn2.Node()}, base+1, target)
	t.NoError(err)

	defer cs.Close()

	cs.reset()

	cs.setState(SyncerPreparing)
	t.NoError(cs.headAndTailManifests())

	{
		b, found, err := cs.storage().Manifest(base + 1)
		t.True(found)
		t.NoError(err)
		t.Equal(base+1, b.Height())
	}

	{
		b, found, err := cs.storage().Manifest(target)
		t.True(found)
		t.NoError(err)
		t.Equal(base+5, b.Height())
	}

	{
		b := cs.TailManifest()
		t.NotNil(b)
		t.Equal(base+5, b.Height())
	}
}

// TestFillManifests setups 4 nodes and 3 nodes has higher blocks rather
// than 1 node.
func (t *testGeneralSyncer) TestFillManifests() {
	ls := t.localstates(4)
	localstate, rn0, rn1, rn2 := ls[0], ls[1], ls[2], ls[3]

	t.setup(localstate, []*Localstate{rn0, rn1, rn2})

	baseBlock := t.lastManifest(localstate.Storage())
	target := baseBlock.Height() + 5
	t.generateBlocks([]*Localstate{rn0, rn1, rn2}, target)

	cs, err := NewGeneralSyncer(localstate, []network.Node{rn0.Node(), rn1.Node(), rn2.Node()}, baseBlock.Height()+1, target)
	t.NoError(err)

	defer cs.Close()

	cs.reset()
	cs.setBaseManifest(baseBlock)
	t.NoError(cs.prepare())

	for i := baseBlock.Height().Int64() + 1; i < target.Int64()+1; i++ {
		b, found, err := cs.storage().Manifest(base.Height(i))
		t.True(found)
		t.NoError(err)

		t.Equal(i, b.Height().Int64())
	}
}

// TestFetchBlocks setups 4 nodes and 3 nodes has higher blocks rather
// than 1 node.
func (t *testGeneralSyncer) TestFetchBlocks() {
	ls := t.localstates(4)
	localstate, rn0, rn1, rn2 := ls[0], ls[1], ls[2], ls[3]

	t.setup(localstate, []*Localstate{localstate, rn0, rn1, rn2})

	baseHeight := t.lastManifest(localstate.Storage()).Height()
	target := baseHeight + 5
	t.generateBlocks([]*Localstate{rn0, rn1, rn2}, target)

	cs, err := NewGeneralSyncer(localstate, []network.Node{rn0.Node(), rn1.Node(), rn2.Node()}, baseHeight+1, target)
	t.NoError(err)

	defer cs.Close()

	cs.reset()

	cs.setState(SyncerPreparing)
	t.NoError(cs.headAndTailManifests())
	t.NoError(cs.fillManifests())

	cs.setState(SyncerSaving)
	t.NoError(cs.startBlocks())

	for i := baseHeight.Int64() + 1; i < target.Int64()+1; i++ {
		b, found, err := cs.storage().Manifest(base.Height(i))
		t.True(found)
		t.NoError(err)

		t.Equal(i, b.Height().Int64())
	}

	for i := baseHeight.Int64() + 1; i < target.Int64()+1; i++ {
		b, found, err := cs.storage().Block(base.Height(i))
		t.True(found)
		t.NoError(err)
		t.Equal(b.Height(), base.Height(i))
	}
}

func (t *testGeneralSyncer) TestSaveBlocks() {
	ls := t.localstates(4)
	localstate, rn0, rn1, rn2 := ls[0], ls[1], ls[2], ls[3]

	t.setup(localstate, []*Localstate{rn0, rn1, rn2})

	baseHeight := t.lastManifest(localstate.Storage()).Height()
	target := baseHeight + 5
	t.generateBlocks([]*Localstate{rn0, rn1, rn2}, target)

	cs, err := NewGeneralSyncer(localstate, []network.Node{rn0.Node(), rn1.Node(), rn2.Node()}, baseHeight+1, target)
	t.NoError(err)

	defer cs.Close()

	cs.reset()

	cs.setState(SyncerPreparing)
	t.NoError(cs.headAndTailManifests())
	t.NoError(cs.fillManifests())
	cs.setState(SyncerPrepared)

	for i := baseHeight.Int64() + 1; i < target.Int64()+1; i++ {
		b, found, err := cs.storage().Manifest(base.Height(i))
		t.True(found)
		t.NoError(err)

		t.Equal(i, b.Height().Int64())
	}

	cs.setState(SyncerSaving)

	t.NoError(cs.startBlocks())

	for i := baseHeight.Int64() + 1; i < target.Int64()+1; i++ {
		b, found, err := cs.storage().Block(base.Height(i))
		t.True(found)
		t.NoError(err)
		t.Equal(b.Height(), base.Height(i))

		_, found, err = localstate.Storage().BlockByHeight(base.Height(i))
		t.False(found)
		t.Nil(err)
	}

	t.NoError(cs.commit())

	for i := baseHeight.Int64() + 1; i < target.Int64()+1; i++ {
		b, found, err := localstate.Storage().BlockByHeight(base.Height(i))
		t.NoError(err)
		t.True(found)
		t.Equal(b.Height(), base.Height(i))
	}
}

func (t *testGeneralSyncer) TestFinishedChan() {
	ls := t.localstates(4)
	localstate, rn0, rn1, rn2 := ls[0], ls[1], ls[2], ls[3]

	t.setup(localstate, []*Localstate{rn0, rn1, rn2})

	baseBlock := t.lastManifest(localstate.Storage())
	target := baseBlock.Height() + 5
	t.generateBlocks([]*Localstate{rn0, rn1, rn2}, target)

	cs, err := NewGeneralSyncer(localstate, []network.Node{rn0.Node(), rn1.Node(), rn2.Node()}, baseBlock.Height()+1, target)
	t.NoError(err)

	defer cs.Close()

	stateChan := make(chan SyncerStateChangedContext)
	finishedChan := make(chan SyncerStateChangedContext)

	go func() {
	end:
		for {
			select {
			case ctx := <-stateChan:
				if ctx.State() != SyncerSaved {
					continue
				}

				finishedChan <- ctx
				break end
			}
		}
	}()

	cs.SetStateChan(stateChan)

	t.NoError(cs.Prepare(baseBlock))

	select {
	case <-time.After(time.Second * 5):
		t.NoError(xerrors.Errorf("timeout to wait to be finished"))
	case ctx := <-finishedChan:
		t.Equal(SyncerSaved, ctx.State())
		t.Equal(baseBlock.Height()+1, ctx.Syncer().HeightFrom())
		t.Equal(target, ctx.Syncer().HeightTo())
	}
}

func (t *testGeneralSyncer) TestFromGenesis() {
	ls := t.localstates(2)
	localstate, _ := ls[0], ls[1]

	t.setup(localstate, nil)

	syncNode := t.emptyLocalstate()
	t.NoError(localstate.Nodes().Add(syncNode.Node()))
	defer t.closeStates(syncNode)

	target := t.lastManifest(localstate.Storage())

	cs, err := NewGeneralSyncer(syncNode, []network.Node{localstate.Node()}, base.PreGenesisHeight, target.Height())
	t.NoError(err)

	defer cs.Close()

	stateChan := make(chan SyncerStateChangedContext)
	finishedChan := make(chan SyncerStateChangedContext)

	go func() {
	end:
		for {
			select {
			case ctx := <-stateChan:
				if ctx.State() != SyncerSaved {
					continue
				}

				finishedChan <- ctx
				break end
			}
		}
	}()

	cs.SetStateChan(stateChan)

	t.NoError(cs.Prepare(nil))

	select {
	case <-time.After(time.Second * 5):
		t.NoError(xerrors.Errorf("timeout to wait to be finished"))
	case ctx := <-finishedChan:
		t.Equal(SyncerSaved, ctx.State())
		t.Equal(base.PreGenesisHeight, ctx.Syncer().HeightFrom())
		t.Equal(target.Height(), ctx.Syncer().HeightTo())
	}
}

func (t *testGeneralSyncer) TestSyncingHandlerFromBallot() {
	ls := t.localstates(4)
	localstate, rn0, rn1, rn2 := ls[0], ls[1], ls[2], ls[3]

	t.setup(localstate, []*Localstate{rn0, rn1, rn2})

	baseBlock := t.lastManifest(localstate.Storage())
	target := baseBlock.Height() + 5
	t.generateBlocks([]*Localstate{rn0, rn1, rn2}, target)

	cs := NewStateSyncingHandler(localstate)

	blt := t.newINITBallot(rn0, base.Round(0), nil)

	t.NoError(cs.Activate(NewStateChangeContext(base.StateJoining, base.StateSyncing, nil, blt)))

	finishedChan := make(chan struct{})
	go func() {
		for {
			b, found, err := localstate.Storage().LastManifest()
			t.NoError(err)
			t.True(found)

			if b.Height() == blt.Height()-1 {
				finishedChan <- struct{}{}
				break
			}

			<-time.After(time.Millisecond * 100)
		}
	}()

	select {
	case <-time.After(time.Second * 10):
		t.NoError(xerrors.Errorf("timeout to wait to be finished"))
		break
	case <-finishedChan:
		break
	}
}

func (t *testGeneralSyncer) TestSyncingHandlerFromINITVoteproof() {
	ls := t.localstates(4)
	localstate, rn0, rn1, rn2 := ls[0], ls[1], ls[2], ls[3]

	t.setup(localstate, []*Localstate{rn0, rn1, rn2})

	baseBlock := t.lastManifest(localstate.Storage())
	target := baseBlock.Height() + 5
	t.generateBlocks([]*Localstate{rn0, rn1, rn2}, target)

	cs := NewStateSyncingHandler(localstate)

	var voteproof base.Voteproof
	{
		b := t.newINITBallot(rn0, base.Round(0), t.lastINITVoteproof(rn0))

		vp, err := t.newVoteproof(b.Stage(), b.INITBallotFactV0, rn0, rn1, rn2)
		t.NoError(err)

		voteproof = vp
	}

	t.NoError(cs.Activate(NewStateChangeContext(base.StateJoining, base.StateSyncing, voteproof, nil)))

	stopChan := make(chan struct{})
	finishedChan := make(chan struct{})
	go func() {
	end:
		for {
			select {
			case <-stopChan:
				break end
			default:
				if t.lastManifest(localstate.Storage()).Height() == voteproof.Height()-1 {
					finishedChan <- struct{}{}
					break end
				}

				<-time.After(time.Millisecond * 10)
			}
		}
	}()

	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("timeout to wait to be finished"))
		stopChan <- struct{}{}
		break
	case <-finishedChan:
		break
	}
}

func (t *testGeneralSyncer) TestSyncingHandlerFromACCEPTVoteproof() {
	ls := t.localstates(4)
	localstate, rn0, rn1, rn2 := ls[0], ls[1], ls[2], ls[3]

	t.setup(localstate, []*Localstate{rn0, rn1, rn2})

	baseBlock := t.lastManifest(localstate.Storage())
	target := baseBlock.Height() + 5
	t.generateBlocks([]*Localstate{rn0, rn1, rn2}, target)

	cs := NewStateSyncingHandler(localstate)

	var voteproof base.Voteproof
	{
		manifest := t.lastManifest(rn0.Storage())
		ab := ballot.NewACCEPTBallotV0(
			rn0.Node().Address(),
			manifest.Height(),
			base.Round(0),
			manifest.Proposal(),
			manifest.Hash(),
			nil,
		)

		vp, err := t.newVoteproof(ab.Stage(), ab.ACCEPTBallotFactV0, rn0, rn1, rn2)
		t.NoError(err)

		voteproof = vp
	}

	t.NoError(cs.Activate(NewStateChangeContext(base.StateJoining, base.StateSyncing, voteproof, nil)))

	stopChan := make(chan struct{})
	finishedChan := make(chan struct{})
	go func() {
	end:
		for {
			select {
			case <-stopChan:
				break end
			default:
				if t.lastManifest(localstate.Storage()).Height() == voteproof.Height() {
					finishedChan <- struct{}{}
					break end
				}

				<-time.After(time.Millisecond * 10)
			}
		}
	}()

	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("timeout to wait to be finished"))
		stopChan <- struct{}{}
		break
	case <-finishedChan:
		break
	}
}

func (t *testGeneralSyncer) TestMissingHead() {
	ls := t.localstates(2)
	localstate, rn0 := ls[0], ls[1]

	t.setup(localstate, []*Localstate{rn0})

	baseBlock := t.lastManifest(localstate.Storage())
	target := baseBlock.Height() + 5
	t.generateBlocks([]*Localstate{rn0}, target)

	head := baseBlock.Height() + 1
	ch := rn0.Node().Channel().(*channetwork.NetworkChanChannel)
	orig := ch.GetManifestsHandler()
	ch.SetGetManifestsHandler(func(heights []base.Height) ([]block.Manifest, error) {
		var bs []block.Manifest
		if l, err := orig(heights); err != nil {
			return nil, err
		} else {
			for _, i := range l {
				if i.Height() == head {
					continue
				}

				bs = append(bs, i)
			}
		}

		return bs, nil
	})

	cs, err := NewGeneralSyncer(localstate, []network.Node{rn0.Node()}, baseBlock.Height()+1, target)
	t.NoError(err)

	defer cs.Close()

	cs.reset()

	err = cs.headAndTailManifests()
	t.Error(err)
}

func (t *testGeneralSyncer) TestMissingTail() {
	ls := t.localstates(2)
	localstate, rn0 := ls[0], ls[1]

	t.setup(localstate, []*Localstate{rn0})

	baseBlock := t.lastManifest(localstate.Storage())
	target := baseBlock.Height() + 5
	t.generateBlocks([]*Localstate{rn0}, target)

	tail := target
	ch := rn0.Node().Channel().(*channetwork.NetworkChanChannel)
	orig := ch.GetManifestsHandler()
	ch.SetGetManifestsHandler(func(heights []base.Height) ([]block.Manifest, error) {
		var bs []block.Manifest
		if l, err := orig(heights); err != nil {
			return nil, err
		} else {
			for _, i := range l {
				if i.Height() == tail {
					continue
				}

				bs = append(bs, i)
			}
		}

		return bs, nil
	})

	cs, err := NewGeneralSyncer(localstate, []network.Node{rn0.Node()}, baseBlock.Height()+1, target)
	t.NoError(err)

	defer cs.Close()

	cs.reset()

	err = cs.headAndTailManifests()
	t.Error(err)
}

func (t *testGeneralSyncer) TestMissingManifests() {
	ls := t.localstates(2)
	localstate, rn0 := ls[0], ls[1]

	t.setup(localstate, []*Localstate{rn0})

	baseBlock := t.lastManifest(localstate.Storage())
	target := baseBlock.Height() + 5
	t.generateBlocks([]*Localstate{rn0}, target)

	missing := target - 1
	ch := rn0.Node().Channel().(*channetwork.NetworkChanChannel)
	orig := ch.GetManifestsHandler()
	ch.SetGetManifestsHandler(func(heights []base.Height) ([]block.Manifest, error) {
		var bs []block.Manifest
		if l, err := orig(heights); err != nil {
			return nil, err
		} else {
			for _, i := range l {
				if i.Height() == missing {
					continue
				}

				bs = append(bs, i)
			}
		}

		return bs, nil
	})

	cs, err := NewGeneralSyncer(localstate, []network.Node{rn0.Node()}, baseBlock.Height()+1, target)
	t.NoError(err)

	defer cs.Close()

	cs.reset()

	err = cs.fillManifests()
	t.Error(err)
}

func (t *testGeneralSyncer) TestMissingBlocks() {
	ls := t.localstates(2)
	localstate, rn0 := ls[0], ls[1]

	t.setup(localstate, []*Localstate{rn0})

	baseBlock := t.lastManifest(localstate.Storage())
	target := baseBlock.Height() + 5
	t.generateBlocks([]*Localstate{rn0}, target)

	missing := target - 1
	ch := rn0.Node().Channel().(*channetwork.NetworkChanChannel)
	orig := ch.GetBlocksHandler()
	ch.SetGetBlocksHandler(func(heights []base.Height) ([]block.Block, error) {
		var bs []block.Block
		if l, err := orig(heights); err != nil {
			return nil, err
		} else {
			for _, i := range l {
				if i.Height() == missing {
					continue
				}

				bs = append(bs, i)
			}
		}

		return bs, nil
	})

	cs, err := NewGeneralSyncer(localstate, []network.Node{rn0.Node()}, baseBlock.Height()+1, target)
	t.NoError(err)

	defer func() {
		if err := cs.Close(); err != nil {
			panic(err)
		}
	}()

	cs.reset()

	t.NoError(cs.Prepare(baseBlock))

	err = cs.fetchBlocksByNodes()
	t.Error(err)
}

func (t *testGeneralSyncer) buildDifferentBlocks(local, remote *Localstate, from, to base.Height) {
	_ = local.Storage().Clean()
	_ = remote.Storage().Clean()
	if from > base.PreGenesisHeight+1 {
		t.generateBlocks([]*Localstate{local, remote}, from-1)
	}

	t.generateBlocks([]*Localstate{local}, to)
	t.generateBlocks([]*Localstate{remote}, to)
}

func (t *testGeneralSyncer) TestRollbackFindUnmatched() {
	cases := []struct {
		name     string
		from     int64
		to       int64
		expected int64
		err      string
	}{
		{
			name:     "genesis unmatched",
			from:     -1,
			to:       5,
			expected: -1,
		},
		{
			name:     "closed unmatched",
			from:     5,
			to:       8,
			expected: 5,
		},
		{
			name:     "inside",
			from:     7,
			to:       22,
			expected: 7,
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				ls := t.localstates(2)
				local, remote := ls[0], ls[1]

				t.setup(local, []*Localstate{remote})

				from, to := base.Height(c.from), base.Height(c.to)
				t.buildDifferentBlocks(local, remote, from, to)

				cs, err := NewGeneralSyncer(local, []network.Node{remote.Node()}, to+1, to+2)
				t.NoError(err, "%d: %v", i, c.name)

				defer cs.Close()

				unmatched, err := cs.compareBlocks(to)
				t.NoError(err, "%d: %v", i, c.name)
				t.Equal(
					c.expected, unmatched.Int64(),
					"%d: %v: %v - %v; %v != %v", i, c.name, c.to, c.from, c.expected, unmatched,
				)

				if c.expected != unmatched.Int64() {
					panic(xerrors.Errorf("failed"))
				}
			},
		)
	}
}

func (t *testGeneralSyncer) TestRollbackWrongGenesisBlocks() {
	ls := t.localstates(2)
	localstate, rn0 := ls[0], ls[1]

	t.setup(localstate, []*Localstate{rn0})

	baseBlock := t.lastManifest(localstate.Storage())

	t.generateBlocks([]*Localstate{localstate}, baseBlock.Height()+3)

	target := baseBlock.Height() + 5

	// NOTE build new blocks from genesis
	bg, err := NewDummyBlocksV0Generator(rn0, target, t.suffrage(rn0, rn0), []*Localstate{rn0})
	t.NoError(err)
	t.NoError(bg.Generate(true))

	fromManifest := t.lastManifest(localstate.Storage())
	cs, err := NewGeneralSyncer(localstate, []network.Node{rn0.Node()}, fromManifest.Height()+1, target)
	t.NoError(err)

	defer cs.Close()

	cs.reset()
	t.NoError(cs.setBaseManifest(fromManifest))

	cs.setState(SyncerPreparing)

	err = cs.headAndTailManifests()
	t.True(xerrors.Is(err, blockIntegrityError))

	var rollbackCtx *BlockIntegrityError
	t.True(xerrors.As(err, &rollbackCtx))
	t.Equal(baseBlock.Height()+3, rollbackCtx.From.Height())
}

func (t *testGeneralSyncer) TestRollbackDetect() {
	ls := t.localstates(2)
	localstate, rn0 := ls[0], ls[1]

	t.setup(localstate, []*Localstate{rn0})

	baseBlock := t.lastManifest(localstate.Storage())

	t.generateBlocks([]*Localstate{localstate}, baseBlock.Height()+3)

	target := baseBlock.Height() + 5
	t.generateBlocks([]*Localstate{rn0}, target)

	fromManifest := t.lastManifest(localstate.Storage())
	cs, err := NewGeneralSyncer(localstate, []network.Node{rn0.Node()}, fromManifest.Height()+1, target)
	t.NoError(err)

	defer cs.Close()

	cs.reset()
	t.NoError(cs.setBaseManifest(fromManifest))

	err = cs.prepare()
	t.True(xerrors.Is(err, blockIntegrityError))
}

func (t *testGeneralSyncer) TestRollback() {
	ls := t.localstates(2)
	local, remote := ls[0], ls[1]

	t.setup(local, []*Localstate{remote})

	baseBlock := t.lastManifest(local.Storage())

	t.generateBlocks([]*Localstate{local}, baseBlock.Height()+3)

	target := baseBlock.Height() + 5
	t.generateBlocks([]*Localstate{remote}, target)

	fromManifest := t.lastManifest(local.Storage())
	cs, err := NewGeneralSyncer(local, []network.Node{remote.Node()}, fromManifest.Height()+1, target)
	t.NoError(err)

	defer cs.Close()

	stateChan := make(chan SyncerStateChangedContext)
	cs.SetStateChan(stateChan)

	t.NoError(cs.Prepare(fromManifest))

end:
	for {
		select {
		case <-time.After(time.Second * 10):
			t.NoError(xerrors.Errorf("timeout to wait to be finished"))

			return
		case ctx := <-stateChan:
			if ctx.State() != SyncerSaved {
				continue
			}
			break end
		}
	}

	lm, _, err := local.Storage().LastManifest()
	t.NoError(err)
	t.Equal(target, lm.Height())

	rm, _, err := remote.Storage().LastManifest()
	t.NoError(err)

	for i := base.PreGenesisHeight; i <= rm.Height(); i++ {
		l, _, err := local.Storage().ManifestByHeight(base.Height(i))
		t.NoError(err)
		r, _, err := remote.Storage().ManifestByHeight(base.Height(i))
		t.NoError(err)

		t.compareManifest(r, l)
	}
}

func TestGeneralSyncer(t *testing.T) {
	suite.Run(t, new(testGeneralSyncer))
}
