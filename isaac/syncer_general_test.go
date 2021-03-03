package isaac

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/network"
	channetwork "github.com/spikeekips/mitum/network/gochan"
)

type testGeneralSyncer struct {
	BaseTest
}

func (t *testGeneralSyncer) TestInvalidFrom() {
	ls := t.Locals(2)
	local, remote := ls[0], ls[1]

	base := t.LastManifest(local.Storage()).Height()
	{ // lower than base
		_, err := NewGeneralSyncer(local.Node(), local.Storage(), local.BlockFS(), local.Policy(),
			[]network.Node{remote.Node()}, base-1, base+2)
		t.Contains(err.Error(), "lower than last block")
	}

	{ // same with base
		_, err := NewGeneralSyncer(local.Node(), local.Storage(), local.BlockFS(), local.Policy(),
			[]network.Node{remote.Node()}, base, base+2)
		t.Contains(err.Error(), "same or lower than last block")
	}

	{ // higher than to
		_, err := NewGeneralSyncer(local.Node(), local.Storage(), local.BlockFS(), local.Policy(),
			[]network.Node{remote.Node()}, base+3, base+2)
		t.Contains(err.Error(), "higher than to height")
	}
}

func (t *testGeneralSyncer) TestInvalidSourceNodes() {
	ls := t.Locals(2)
	local, _ := ls[0], ls[1]

	t.SetupNodes(local, nil)

	base := t.LastManifest(local.Storage()).Height()

	{ // nil node
		_, err := NewGeneralSyncer(local.Node(), local.Storage(), local.BlockFS(), local.Policy(),
			nil, base+1, base+2)
		t.Contains(err.Error(), "empty source nodes")
	}

	{ // same with local node
		_, err := NewGeneralSyncer(local.Node(), local.Storage(), local.BlockFS(), local.Policy(),
			[]network.Node{local.Node()}, base+1, base+2)
		t.Contains(err.Error(), "same with local node")
	}
}

func (t *testGeneralSyncer) TestNew() {
	ls := t.Locals(2)
	local, remote := ls[0], ls[1]

	t.SetupNodes(local, []*Local{remote})

	target := t.LastManifest(local.Storage()).Height() + 1
	t.GenerateBlocks([]*Local{remote}, target)

	cs, err := NewGeneralSyncer(local.Node(), local.Storage(), local.BlockFS(), local.Policy(),
		[]network.Node{remote.Node()}, target, target)
	t.NoError(err)

	defer cs.Close()

	_ = (interface{})(cs).(Syncer)
	t.Implements((*Syncer)(nil), cs)

	t.Equal(SyncerCreated, cs.State())
}

// TestHeadAndTailManifests setups 4 nodes and 3 nodes has higher blocks rather
// than 1 node.
func (t *testGeneralSyncer) TestHeadAndTailManifests() {
	ls := t.Locals(4)
	local, rn0, rn1, rn2 := ls[0], ls[1], ls[2], ls[3]

	t.SetupNodes(local, []*Local{rn0, rn1, rn2})

	base := t.LastManifest(local.Storage()).Height()
	target := base + 5
	t.GenerateBlocks([]*Local{rn0, rn1, rn2}, target)

	cs, err := NewGeneralSyncer(local.Node(), local.Storage(), local.BlockFS(), local.Policy(),
		[]network.Node{rn0.Node(), rn1.Node(), rn2.Node()}, base+1, target)
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
	ls := t.Locals(4)
	local, rn0, rn1, rn2 := ls[0], ls[1], ls[2], ls[3]

	t.SetupNodes(local, []*Local{rn0, rn1, rn2})

	baseBlock := t.LastManifest(local.Storage())
	target := baseBlock.Height() + 5
	t.GenerateBlocks([]*Local{rn0, rn1, rn2}, target)

	cs, err := NewGeneralSyncer(local.Node(), local.Storage(), local.BlockFS(), local.Policy(),
		[]network.Node{rn0.Node(), rn1.Node(), rn2.Node()}, baseBlock.Height()+1, target)
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
	ls := t.Locals(4)
	local, rn0, rn1, rn2 := ls[0], ls[1], ls[2], ls[3]

	t.SetupNodes(local, []*Local{local, rn0, rn1, rn2})

	baseHeight := t.LastManifest(local.Storage()).Height()
	target := baseHeight + 5
	t.GenerateBlocks([]*Local{rn0, rn1, rn2}, target)

	cs, err := NewGeneralSyncer(local.Node(), local.Storage(), local.BlockFS(), local.Policy(),
		[]network.Node{rn0.Node(), rn1.Node(), rn2.Node()}, baseHeight+1, target)
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

	for i := baseHeight + 1; i < target+1; i++ {
		var b block.Block
		for _, j := range cs.blocks {
			if j.Height() == i {
				b = j
				break
			}
		}

		t.NotNil(b)
	}
}

func (t *testGeneralSyncer) TestFetchBlocksButSomeNodesFailed() {
	ls := t.Locals(4)
	local, rn0, rn1, rn2 := ls[0], ls[1], ls[2], ls[3]

	t.SetupNodes(local, []*Local{local, rn0, rn1, rn2})

	_ = local.Policy().SetThresholdRatio(base.ThresholdRatio(100))

	baseHeight := t.LastManifest(local.Storage()).Height()
	target := baseHeight + 3
	t.GenerateBlocks([]*Local{rn0, rn1, rn2}, target)

	// only one node, rn0 will return correct manifest
	for i := range ls[2:] {
		ch := ls[i+2].Node().Channel().(*channetwork.Channel)
		orig := ch.GetManifestsHandler()
		ch.SetGetManifestsHandler(func(heights []base.Height) ([]block.Manifest, error) {
			var bs []block.Manifest
			if l, err := orig(heights); err != nil {
				return nil, err
			} else {
				for _, i := range l {
					if i.Height() == target {
						continue
					}

					bs = append(bs, i)
				}
			}

			return bs, nil
		})
	}

	cs, err := NewGeneralSyncer(local.Node(), local.Storage(), local.BlockFS(), local.Policy(),
		[]network.Node{rn0.Node(), rn1.Node(), rn2.Node()}, baseHeight+1, target)
	t.NoError(err)

	defer cs.Close()

	cs.reset()

	cs.setState(SyncerPreparing)
	t.NoError(cs.headAndTailManifests())
}

func (t *testGeneralSyncer) TestFetchBlocksButAllNodesFailed() {
	ls := t.Locals(4)
	local, rn0, rn1, rn2 := ls[0], ls[1], ls[2], ls[3]

	t.SetupNodes(local, []*Local{local, rn0, rn1, rn2})

	_ = local.Policy().SetThresholdRatio(base.ThresholdRatio(100))

	baseHeight := t.LastManifest(local.Storage()).Height()
	target := baseHeight + 3
	t.GenerateBlocks([]*Local{rn0, rn1, rn2}, target)

	for i := range ls[1:] {
		ch := ls[i+1].Node().Channel().(*channetwork.Channel)
		orig := ch.GetManifestsHandler()
		ch.SetGetManifestsHandler(func(heights []base.Height) ([]block.Manifest, error) {
			var bs []block.Manifest
			if l, err := orig(heights); err != nil {
				return nil, err
			} else {
				for _, i := range l {
					if i.Height() == target {
						continue
					}

					bs = append(bs, i)
				}
			}

			return bs, nil
		})
	}

	cs, err := NewGeneralSyncer(local.Node(), local.Storage(), local.BlockFS(), local.Policy(),
		[]network.Node{rn0.Node(), rn1.Node(), rn2.Node()}, baseHeight+1, target)
	t.NoError(err)

	defer cs.Close()

	cs.reset()

	cs.setState(SyncerPreparing)
	err = cs.headAndTailManifests()
	t.Contains(err.Error(), "nothing fetched")
}

func (t *testGeneralSyncer) TestSaveBlocks() {
	ls := t.Locals(4)
	local, rn0, rn1, rn2 := ls[0], ls[1], ls[2], ls[3]

	t.SetupNodes(local, []*Local{rn0, rn1, rn2})

	baseHeight := t.LastManifest(local.Storage()).Height()
	target := baseHeight + 5
	t.GenerateBlocks([]*Local{rn0, rn1, rn2}, target)

	cs, err := NewGeneralSyncer(local.Node(), local.Storage(), local.BlockFS(), local.Policy(),
		[]network.Node{rn0.Node(), rn1.Node(), rn2.Node()}, baseHeight+1, target)
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

	for i := baseHeight + 1; i < target+1; i++ {
		var b block.Block
		for _, j := range cs.blocks {
			if j.Height() == i {
				b = j
				break
			}
		}

		t.NotNil(b)
	}

	t.NoError(cs.commit())
	t.NoError(cs.saveBlockFS())

	for i := baseHeight.Int64() + 1; i < target.Int64()+1; i++ {
		b, err := local.BlockFS().Load(base.Height(i))
		t.NoError(err)
		t.Equal(b.Height(), base.Height(i))
	}
}

func (t *testGeneralSyncer) TestFinishedChan() {
	ls := t.Locals(4)
	local, rn0, rn1, rn2 := ls[0], ls[1], ls[2], ls[3]

	t.SetupNodes(local, []*Local{rn0, rn1, rn2})

	baseBlock := t.LastManifest(local.Storage())
	target := baseBlock.Height() + 5
	t.GenerateBlocks([]*Local{rn0, rn1, rn2}, target)

	cs, err := NewGeneralSyncer(local.Node(), local.Storage(), local.BlockFS(), local.Policy(),
		[]network.Node{rn0.Node(), rn1.Node(), rn2.Node()}, baseBlock.Height()+1, target)
	t.NoError(err)

	defer cs.Close()

	stateChan := make(chan SyncerStateChangedContext)
	finishedChan := make(chan SyncerStateChangedContext)

	go func() {
		for ctx := range stateChan {
			if ctx.State() != SyncerSaved {
				continue
			}

			finishedChan <- ctx
			break
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
	ls := t.Locals(2)
	local, _ := ls[0], ls[1]

	t.SetupNodes(local, nil)

	syncNode := t.EmptyLocal()
	t.NoError(local.Nodes().Add(syncNode.Node()))
	defer t.CloseStates(syncNode)

	target := t.LastManifest(local.Storage())

	cs, err := NewGeneralSyncer(syncNode.Node(), syncNode.Storage(), syncNode.BlockFS(), syncNode.Policy(), []network.Node{local.Node()}, base.PreGenesisHeight, target.Height())
	t.NoError(err)

	defer cs.Close()

	stateChan := make(chan SyncerStateChangedContext)
	finishedChan := make(chan SyncerStateChangedContext)

	go func() {
		for ctx := range stateChan {
			if ctx.State() != SyncerSaved {
				continue
			}

			finishedChan <- ctx
			break
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

func (t *testGeneralSyncer) TestMissingHead() {
	ls := t.Locals(2)
	local, rn0 := ls[0], ls[1]

	t.SetupNodes(local, []*Local{rn0})

	baseBlock := t.LastManifest(local.Storage())
	target := baseBlock.Height() + 5
	t.GenerateBlocks([]*Local{rn0}, target)

	head := baseBlock.Height() + 1
	ch := rn0.Node().Channel().(*channetwork.Channel)
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

	cs, err := NewGeneralSyncer(local.Node(), local.Storage(), local.BlockFS(), local.Policy(),
		[]network.Node{rn0.Node()}, baseBlock.Height()+1, target)
	t.NoError(err)

	defer cs.Close()

	cs.reset()

	cs.setState(SyncerPreparing)
	t.Error(cs.headAndTailManifests())
}

func (t *testGeneralSyncer) TestMissingTail() {
	ls := t.Locals(2)
	local, rn0 := ls[0], ls[1]

	t.SetupNodes(local, []*Local{rn0})

	baseBlock := t.LastManifest(local.Storage())
	target := baseBlock.Height() + 5
	t.GenerateBlocks([]*Local{rn0}, target)

	tail := target
	ch := rn0.Node().Channel().(*channetwork.Channel)
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

	cs, err := NewGeneralSyncer(local.Node(), local.Storage(), local.BlockFS(), local.Policy(),
		[]network.Node{rn0.Node()}, baseBlock.Height()+1, target)
	t.NoError(err)

	defer cs.Close()

	cs.reset()

	cs.setState(SyncerPreparing)
	t.Error(cs.headAndTailManifests())
}

func (t *testGeneralSyncer) TestMissingManifests() {
	ls := t.Locals(2)
	local, rn0 := ls[0], ls[1]

	t.SetupNodes(local, []*Local{rn0})

	baseBlock := t.LastManifest(local.Storage())
	target := baseBlock.Height() + 5
	t.GenerateBlocks([]*Local{rn0}, target)

	missing := target - 1
	ch := rn0.Node().Channel().(*channetwork.Channel)
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

	cs, err := NewGeneralSyncer(local.Node(), local.Storage(), local.BlockFS(), local.Policy(),
		[]network.Node{rn0.Node()}, baseBlock.Height()+1, target)
	t.NoError(err)

	defer cs.Close()

	cs.reset()

	cs.setState(SyncerPreparing)
	t.Error(cs.fillManifests())
}

func (t *testGeneralSyncer) TestMissingBlocks() {
	ls := t.Locals(2)
	local, rn0 := ls[0], ls[1]

	t.SetupNodes(local, []*Local{rn0})

	baseBlock := t.LastManifest(local.Storage())
	target := baseBlock.Height() + 5
	t.GenerateBlocks([]*Local{rn0}, target)

	missing := target - 1
	ch := rn0.Node().Channel().(*channetwork.Channel)
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

	cs, err := NewGeneralSyncer(local.Node(), local.Storage(), local.BlockFS(), local.Policy(),
		[]network.Node{rn0.Node()}, baseBlock.Height()+1, target)
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

func (t *testGeneralSyncer) buildDifferentBlocks(local, remote *Local, from, to base.Height) {
	_ = local.Storage().Clean()
	_ = local.BlockFS().Clean(false)
	_ = remote.Storage().Clean()
	_ = remote.BlockFS().Clean(false)
	if from > base.PreGenesisHeight+1 {
		t.GenerateBlocks([]*Local{local, remote}, from-1)
	}

	t.GenerateBlocks([]*Local{local}, to)
	t.GenerateBlocks([]*Local{remote}, to)
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
				ls := t.Locals(2)
				local, remote := ls[0], ls[1]

				t.SetupNodes(local, []*Local{remote})

				from, to := base.Height(c.from), base.Height(c.to)
				t.buildDifferentBlocks(local, remote, from, to)

				cs, err := NewGeneralSyncer(local.Node(), local.Storage(), local.BlockFS(), local.Policy(),
					[]network.Node{remote.Node()}, to+1, to+2)
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
	ls := t.Locals(2)
	local, rn0 := ls[0], ls[1]

	t.SetupNodes(local, []*Local{rn0})

	baseBlock := t.LastManifest(local.Storage())

	t.GenerateBlocks([]*Local{local}, baseBlock.Height()+3)

	target := baseBlock.Height() + 5

	// NOTE build new blocks from genesis
	bg, err := NewDummyBlocksV0Generator(rn0, target, t.Suffrage(rn0, rn0), []*Local{rn0})
	t.NoError(err)
	t.NoError(bg.Generate(true))

	fromManifest := t.LastManifest(local.Storage())
	cs, err := NewGeneralSyncer(local.Node(), local.Storage(), local.BlockFS(), local.Policy(),
		[]network.Node{rn0.Node()}, fromManifest.Height()+1, target)
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
	ls := t.Locals(2)
	local, rn0 := ls[0], ls[1]

	t.SetupNodes(local, []*Local{rn0})

	baseBlock := t.LastManifest(local.Storage())

	t.GenerateBlocks([]*Local{local}, baseBlock.Height()+3)

	target := baseBlock.Height() + 5
	t.GenerateBlocks([]*Local{rn0}, target)

	fromManifest := t.LastManifest(local.Storage())
	cs, err := NewGeneralSyncer(local.Node(), local.Storage(), local.BlockFS(), local.Policy(),
		[]network.Node{rn0.Node()}, fromManifest.Height()+1, target)
	t.NoError(err)

	defer cs.Close()

	cs.reset()
	t.NoError(cs.setBaseManifest(fromManifest))

	err = cs.prepare()
	t.True(xerrors.Is(err, blockIntegrityError))
}

func (t *testGeneralSyncer) TestRollback() {
	ls := t.Locals(2)
	local, remote := ls[0], ls[1]

	t.SetupNodes(local, []*Local{remote})

	baseBlock := t.LastManifest(local.Storage())

	t.GenerateBlocks([]*Local{local}, baseBlock.Height()+3)

	target := baseBlock.Height() + 5
	t.GenerateBlocks([]*Local{remote}, target)

	fromManifest := t.LastManifest(local.Storage())
	cs, err := NewGeneralSyncer(local.Node(), local.Storage(), local.BlockFS(), local.Policy(),
		[]network.Node{remote.Node()}, fromManifest.Height()+1, target)
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

		t.CompareManifest(r, l)
	}
}

func TestGeneralSyncer(t *testing.T) {
	suite.Run(t, new(testGeneralSyncer))
}
