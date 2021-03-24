package isaac

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/network"
	channetwork "github.com/spikeekips/mitum/network/gochan"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata/localfs"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testGeneralSyncer struct {
	BaseTest
}

func (t *testGeneralSyncer) TestInvalidFrom() {
	ls := t.Locals(2)
	local, remote := ls[0], ls[1]

	bm := t.LastManifest(local.Database())
	lower, found, err := local.Database().ManifestByHeight(bm.Height() - 1)
	t.NoError(err)
	t.True(found)

	base := bm.Height()
	{ // lower than base
		_, err := NewGeneralSyncer(local.Node(), local.Database(), local.BlockData(), local.Policy(),
			[]network.Node{remote.Node()}, lower, base+2)
		t.Contains(err.Error(), "lower than last block")
	}

	{ // higher than to
		_, err := NewGeneralSyncer(local.Node(), local.Database(), local.BlockData(), local.Policy(),
			[]network.Node{remote.Node()}, bm, bm.Height()-2)
		t.Contains(err.Error(), "greater than to height")
	}
}

func (t *testGeneralSyncer) TestInvalidSourceNodes() {
	ls := t.Locals(2)
	local, _ := ls[0], ls[1]

	t.SetupNodes(local, nil)

	bm := t.LastManifest(local.Database())
	base := bm.Height()

	{ // nil node
		_, err := NewGeneralSyncer(local.Node(), local.Database(), local.BlockData(), local.Policy(),
			nil, bm, base+2)
		t.Contains(err.Error(), "empty source nodes")
	}

	{ // same with local node
		_, err := NewGeneralSyncer(local.Node(), local.Database(), local.BlockData(), local.Policy(),
			[]network.Node{local.Node()}, bm, base+2)
		t.Contains(err.Error(), "same with local node")
	}
}

func (t *testGeneralSyncer) TestNew() {
	ls := t.Locals(2)
	local, remote := ls[0], ls[1]

	t.SetupNodes(local, []*Local{remote})

	bm := t.LastManifest(local.Database())
	target := bm.Height() + 1
	t.GenerateBlocks([]*Local{remote}, target)

	cs, err := NewGeneralSyncer(local.Node(), local.Database(), local.BlockData(), local.Policy(),
		[]network.Node{remote.Node()}, bm, target)
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

	bm := t.LastManifest(local.Database())
	base := bm.Height()
	target := base + 5
	t.GenerateBlocks([]*Local{rn0, rn1, rn2}, target)

	cs, err := NewGeneralSyncer(local.Node(), local.Database(), local.BlockData(), local.Policy(),
		[]network.Node{rn0.Node(), rn1.Node(), rn2.Node()}, bm, target)
	t.NoError(err)
	defer cs.Close()

	cs.reset()

	cs.setState(SyncerPreparing, false)
	t.NoError(cs.headAndTailManifests())

	{
		b, found, err := cs.database().Manifest(base + 1)
		t.True(found)
		t.NoError(err)
		t.Equal(base+1, b.Height())
	}

	{
		b, found, err := cs.database().Manifest(target)
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

	baseBlock := t.LastManifest(local.Database())
	target := baseBlock.Height() + 5
	t.GenerateBlocks([]*Local{rn0, rn1, rn2}, target)

	cs, err := NewGeneralSyncer(local.Node(), local.Database(), local.BlockData(), local.Policy(),
		[]network.Node{rn0.Node(), rn1.Node(), rn2.Node()}, baseBlock, target)
	t.NoError(err)
	defer cs.Close()

	cs.reset()
	cs.setState(SyncerPreparing, false)
	t.NoError(cs.headAndTailManifests())
	t.NoError(cs.fillManifests())

	for i := baseBlock.Height().Int64() + 1; i < target.Int64()+1; i++ {
		b, found, err := cs.database().Manifest(base.Height(i))
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

	bm := t.LastManifest(local.Database())
	baseHeight := bm.Height()
	target := baseHeight + 5
	t.GenerateBlocks([]*Local{rn0, rn1, rn2}, target)

	cs, err := NewGeneralSyncer(local.Node(), local.Database(), local.BlockData(), local.Policy(),
		[]network.Node{rn0.Node(), rn1.Node(), rn2.Node()}, bm, target)
	t.NoError(err)
	defer cs.Close()

	cs.reset()

	cs.setState(SyncerPreparing, false)
	t.NoError(cs.headAndTailManifests())
	t.NoError(cs.fillManifests())

	cs.setState(SyncerSaving, false)
	t.NoError(cs.startBlocks())

	for i := baseHeight.Int64() + 1; i < target.Int64()+1; i++ {
		b, found, err := cs.database().Manifest(base.Height(i))
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

	bm := t.LastManifest(local.Database())
	baseHeight := bm.Height()
	target := baseHeight + 3
	t.GenerateBlocks([]*Local{rn0, rn1, rn2}, target)

	// only one node, rn0 will return correct manifest
	for i := range ls[2:] {
		ch := ls[i+2].Node().Channel().(*channetwork.Channel)

		orig := ch.GetBlockDataHandler()
		ch.SetBlockDataHandler(func(p string) (io.ReadCloser, func() error, error) {
			bp := filepath.Base(p)

			if !strings.Contains(p, "manifest") {
				return orig(p)
			}

			if strings.HasPrefix(bp, fmt.Sprintf("%d-", target)) {
				return nil, nil, nil
			} else {
				return orig(p)
			}
		})

	}

	cs, err := NewGeneralSyncer(local.Node(), local.Database(), local.BlockData(), local.Policy(),
		[]network.Node{rn0.Node(), rn1.Node(), rn2.Node()}, bm, target)
	t.NoError(err)
	defer cs.Close()

	cs.reset()

	cs.setState(SyncerPreparing, false)
	t.NoError(cs.headAndTailManifests())
}

func (t *testGeneralSyncer) TestFetchBlocksButAllNodesFailed() {
	ls := t.Locals(4)
	local, rn0, rn1, rn2 := ls[0], ls[1], ls[2], ls[3]

	t.SetupNodes(local, []*Local{local, rn0, rn1, rn2})

	_ = local.Policy().SetThresholdRatio(base.ThresholdRatio(100))

	bm := t.LastManifest(local.Database())
	baseHeight := bm.Height()
	target := baseHeight + 3
	t.GenerateBlocks([]*Local{rn0, rn1, rn2}, target)

	for i := range ls[1:] {
		ch := ls[i+1].Node().Channel().(*channetwork.Channel)
		orig := ch.GetBlockDataMapsHandler()
		ch.SetBlockDataMapsHandler(func(heights []base.Height) ([]block.BlockDataMap, error) {
			var bds []block.BlockDataMap
			if l, err := orig(heights); err != nil {
				return nil, err
			} else {
				for _, i := range l {
					if i.Height() == target {
						continue
					}

					bds = append(bds, i)
				}
			}

			return bds, nil
		})
	}

	cs, err := NewGeneralSyncer(local.Node(), local.Database(), local.BlockData(), local.Policy(),
		[]network.Node{rn0.Node(), rn1.Node(), rn2.Node()}, bm, target)
	t.NoError(err)
	defer cs.Close()

	cs.reset()

	cs.setState(SyncerPreparing, false)
	err = cs.headAndTailManifests()
	t.Contains(err.Error(), "nothing fetched")
}

func (t *testGeneralSyncer) TestSaveBlocks() {
	ls := t.Locals(4)
	local, rn0, rn1, rn2 := ls[0], ls[1], ls[2], ls[3]

	t.SetupNodes(local, []*Local{rn0, rn1, rn2})

	bm := t.LastManifest(local.Database())
	baseHeight := bm.Height()
	target := baseHeight + 5
	t.GenerateBlocks([]*Local{rn0, rn1, rn2}, target)

	cs, err := NewGeneralSyncer(local.Node(), local.Database(), local.BlockData(), local.Policy(),
		[]network.Node{rn0.Node(), rn1.Node(), rn2.Node()}, bm, target)
	t.NoError(err)
	defer cs.Close()

	cs.reset()

	cs.setState(SyncerPreparing, false)
	t.NoError(cs.headAndTailManifests())
	t.NoError(cs.fillManifests())
	cs.setState(SyncerPrepared, false)

	for i := baseHeight.Int64() + 1; i < target.Int64()+1; i++ {
		b, found, err := cs.database().Manifest(base.Height(i))
		t.True(found)
		t.NoError(err)

		t.Equal(i, b.Height().Int64())
	}

	cs.setState(SyncerSaving, false)

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

	for i := baseHeight + 1; i < target+1; i++ {
		t.True(local.BlockData().Exists(i))

		blk, err := localfs.LoadBlock(local.BlockData().(*localfs.BlockData), i)
		t.NoError(err)
		t.Equal(blk.Height(), i)
	}
}

func (t *testGeneralSyncer) TestFinishedChan() {
	ls := t.Locals(4)
	local, rn0, rn1, rn2 := ls[0], ls[1], ls[2], ls[3]

	t.SetupNodes(local, []*Local{rn0, rn1, rn2})

	baseBlock := t.LastManifest(local.Database())
	target := baseBlock.Height() + 5
	t.GenerateBlocks([]*Local{rn0, rn1, rn2}, target)

	cs, err := NewGeneralSyncer(local.Node(), local.Database(), local.BlockData(), local.Policy(),
		[]network.Node{rn0.Node(), rn1.Node(), rn2.Node()}, baseBlock, target)
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

	t.NoError(cs.Prepare())

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

	target := t.LastManifest(local.Database())

	cs, err := NewGeneralSyncer(syncNode.Node(), syncNode.Database(), syncNode.BlockData(), syncNode.Policy(), []network.Node{local.Node()}, nil, target.Height())
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

	t.NoError(cs.Prepare())

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

	baseBlock := t.LastManifest(local.Database())
	target := baseBlock.Height() + 5
	t.GenerateBlocks([]*Local{rn0}, target)

	head := baseBlock.Height() + 1
	ch := rn0.Node().Channel().(*channetwork.Channel)
	orig := ch.GetBlockDataMapsHandler()
	ch.SetBlockDataMapsHandler(func(heights []base.Height) ([]block.BlockDataMap, error) {
		var bds []block.BlockDataMap
		if l, err := orig(heights); err != nil {
			return nil, err
		} else {
			for _, i := range l {
				if i.Height() == head {
					continue
				}

				bds = append(bds, i)
			}
		}

		return bds, nil
	})

	cs, err := NewGeneralSyncer(local.Node(), local.Database(), local.BlockData(), local.Policy(),
		[]network.Node{rn0.Node()}, baseBlock, target)
	t.NoError(err)
	defer cs.Close()

	cs.reset()

	cs.setState(SyncerPreparing, false)
	t.Error(cs.headAndTailManifests())
}

func (t *testGeneralSyncer) TestMissingTail() {
	ls := t.Locals(2)
	local, rn0 := ls[0], ls[1]

	t.SetupNodes(local, []*Local{rn0})

	baseBlock := t.LastManifest(local.Database())
	target := baseBlock.Height() + 5
	t.GenerateBlocks([]*Local{rn0}, target)

	tail := target
	ch := rn0.Node().Channel().(*channetwork.Channel)

	orig := ch.GetBlockDataMapsHandler()
	ch.SetBlockDataMapsHandler(func(heights []base.Height) ([]block.BlockDataMap, error) {
		var bds []block.BlockDataMap
		if l, err := orig(heights); err != nil {
			return nil, err
		} else {
			for _, i := range l {
				if i.Height() == tail {
					continue
				}

				bds = append(bds, i)
			}
		}

		return bds, nil
	})

	cs, err := NewGeneralSyncer(local.Node(), local.Database(), local.BlockData(), local.Policy(),
		[]network.Node{rn0.Node()}, baseBlock, target)
	t.NoError(err)
	defer cs.Close()

	cs.reset()

	cs.setState(SyncerPreparing, false)
	t.Error(cs.headAndTailManifests())
}

func (t *testGeneralSyncer) TestMissingManifests() {
	ls := t.Locals(2)
	local, rn0 := ls[0], ls[1]

	t.SetupNodes(local, []*Local{rn0})

	baseBlock := t.LastManifest(local.Database())
	target := baseBlock.Height() + 5
	t.GenerateBlocks([]*Local{rn0}, target)

	missing := target - 1
	ch := rn0.Node().Channel().(*channetwork.Channel)
	orig := ch.GetBlockDataMapsHandler()
	ch.SetBlockDataMapsHandler(func(heights []base.Height) ([]block.BlockDataMap, error) {
		var bs []block.BlockDataMap
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

	cs, err := NewGeneralSyncer(local.Node(), local.Database(), local.BlockData(), local.Policy(),
		[]network.Node{rn0.Node()}, baseBlock, target)
	t.NoError(err)
	defer cs.Close()

	cs.reset()

	cs.setState(SyncerPreparing, false)
	t.Error(cs.fillManifests())
}

func (t *testGeneralSyncer) TestMissingBlocks() {
	ls := t.Locals(2)
	local, rn0 := ls[0], ls[1]

	t.SetupNodes(local, []*Local{rn0})

	baseBlock := t.LastManifest(local.Database())
	target := baseBlock.Height() + 5
	t.GenerateBlocks([]*Local{rn0}, target)

	missing := target - 1
	ch := rn0.Node().Channel().(*channetwork.Channel)
	orig := ch.GetBlockDataHandler()
	ch.SetBlockDataHandler(func(p string) (io.ReadCloser, func() error, error) {
		bp := filepath.Base(p)

		if strings.Contains(p, "manifest") {
			return orig(p)
		}

		if strings.HasPrefix(bp, fmt.Sprintf("%d-", missing)) {
			if strings.Contains(p, "-operations-") {
				return nil, nil, storage.NotFoundError
			}
		}

		return orig(p)
	})

	cs, err := NewGeneralSyncer(local.Node(), local.Database(), local.BlockData(), local.Policy(),
		[]network.Node{rn0.Node()}, baseBlock, target)
	t.NoError(err)

	defer func() {
		if err := cs.Close(); err != nil {
			panic(err)
		}
	}()

	cs.reset()

	t.NoError(cs.Prepare())

	err = cs.fetchBlocksByNodes()
	t.Error(err)
}

func (t *testGeneralSyncer) buildDifferentBlocks(local, remote *Local, from, to base.Height) {
	_ = local.Database().Clean()
	_ = local.BlockData().Clean(false)
	_ = remote.Database().Clean()
	_ = remote.BlockData().Clean(false)
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

				base, found, err := local.Database().ManifestByHeight(to)
				t.NoError(err)
				t.True(found)

				cs, err := NewGeneralSyncer(local.Node(), local.Database(), local.BlockData(), local.Policy(),
					[]network.Node{remote.Node()}, base, base.Height()+2)
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

	baseBlock := t.LastManifest(local.Database())

	t.GenerateBlocks([]*Local{local}, baseBlock.Height()+3)

	target := baseBlock.Height() + 5

	// NOTE build new blocks from genesis
	bg, err := NewDummyBlocksV0Generator(rn0, target, t.Suffrage(rn0, rn0), []*Local{rn0})
	t.NoError(err)
	t.NoError(bg.Generate(true))

	fromManifest := t.LastManifest(local.Database())
	cs, err := NewGeneralSyncer(local.Node(), local.Database(), local.BlockData(), local.Policy(),
		[]network.Node{rn0.Node()}, fromManifest, target)
	t.NoError(err)
	defer cs.Close()

	cs.reset()

	cs.setState(SyncerPreparing, false)

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

	baseBlock := t.LastManifest(local.Database())

	t.GenerateBlocks([]*Local{local}, baseBlock.Height()+3)

	target := baseBlock.Height() + 5
	t.GenerateBlocks([]*Local{rn0}, target)

	fromManifest := t.LastManifest(local.Database())
	cs, err := NewGeneralSyncer(local.Node(), local.Database(), local.BlockData(), local.Policy(),
		[]network.Node{rn0.Node()}, fromManifest, target)
	t.NoError(err)
	defer cs.Close()

	cs.reset()

	err = cs.prepare()
	t.True(xerrors.Is(err, blockIntegrityError))
}

func (t *testGeneralSyncer) TestRollback() {
	ls := t.Locals(2)
	local, remote := ls[0], ls[1]

	t.SetupNodes(local, []*Local{remote})

	baseBlock := t.LastManifest(local.Database())

	t.GenerateBlocks([]*Local{local}, baseBlock.Height()+3)

	target := baseBlock.Height() + 5
	t.GenerateBlocks([]*Local{remote}, target)

	fromManifest := t.LastManifest(local.Database())
	cs, err := NewGeneralSyncer(local.Node(), local.Database(), local.BlockData(), local.Policy(),
		[]network.Node{remote.Node()}, fromManifest, target)
	t.NoError(err)
	defer cs.Close()

	stateChan := make(chan SyncerStateChangedContext)
	cs.SetStateChan(stateChan)

	t.NoError(cs.Prepare())

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

	lm, _, err := local.Database().LastManifest()
	t.NoError(err)
	t.Equal(target, lm.Height())

	rm, _, err := remote.Database().LastManifest()
	t.NoError(err)

	for i := base.PreGenesisHeight; i <= rm.Height(); i++ {
		l, _, err := local.Database().ManifestByHeight(base.Height(i))
		t.NoError(err)
		r, _, err := remote.Database().ManifestByHeight(base.Height(i))
		t.NoError(err)

		t.CompareManifest(r, l)
	}
}

func TestGeneralSyncer(t *testing.T) {
	suite.Run(t, new(testGeneralSyncer))
}
