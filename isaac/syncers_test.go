package isaac

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/network"
	"github.com/stretchr/testify/suite"
)

func (sy *Syncers) LastSyncer() Syncer {
	sy.RLock()
	defer sy.RUnlock()

	return sy.lastSyncer
}

type testSyncers struct {
	BaseTest
}

func (t *testSyncers) TestNew() {
	ls := t.Locals(2)
	local, remote := ls[0], ls[1]

	t.SetupNodes(local, []*Local{remote})

	baseManifest, found, err := local.Database().LastManifest()
	t.NoError(err)
	t.True(found)

	finishedChan := make(chan base.Height, 10)
	blocksChan := make(chan []block.Block, 10)

	ss := NewSyncers(local.Database(), local.Blockdata(), local.Policy(), baseManifest, func() map[string]network.Channel {
		return map[string]network.Channel{
			remote.Node().String(): remote.Channel(),
		}
	})

	ss.WhenFinished(func(height base.Height) {
		finishedChan <- height
	})
	ss.WhenBlockSaved(func(blocks []block.Block) {
		blocksChan <- blocks
	})
	t.NoError(ss.Start())

	defer ss.Stop()

	fromHeight := t.LastManifest(local.Database()).Height() + 1
	target := fromHeight + 2
	t.True(target < fromHeight+base.Height(int64(ss.limitBlocksPerSyncer)))
	t.GenerateBlocks([]*Local{remote}, target)

	isFinished, err := ss.Add(target, []base.Node{remote.Node()})
	t.NoError(err)
	t.False(isFinished)

	var blocks []base.Height

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

end:
	for {
		select {
		case <-ctx.Done():
			t.NoError(errors.Errorf("timeout to wait to be finished"))

			break end
		case bs := <-blocksChan:
			for _, blk := range bs {
				blocks = append(blocks, blk.Height())
			}
		case height := <-finishedChan:
			if target == height {
				break end
			}
		}
	}

	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i] < blocks[j]
	})

	var expectedBlocks []base.Height
	for i := fromHeight; i <= target; i++ {
		expectedBlocks = append(expectedBlocks, i)
	}

	t.Equal(expectedBlocks, blocks)

	rm, found, err := remote.Database().LastManifest()
	t.NoError(err)
	t.True(found)
	lm, found, err := local.Database().LastManifest()
	t.NoError(err)
	t.True(found)

	t.CompareManifest(rm, lm)
}

func (t *testSyncers) TestMultipleSyncers() {
	ls := t.Locals(2)
	local, remote := ls[0], ls[1]

	t.SetupNodes(local, []*Local{remote})

	target := t.LastManifest(local.Database()).Height() + 10
	t.GenerateBlocks([]*Local{remote}, target)

	baseManifest, found, err := local.Database().LastManifest()
	t.NoError(err)
	t.True(found)

	finishedChan := make(chan base.Height, 10)

	ss := NewSyncers(local.Database(), local.Blockdata(), local.Policy(), baseManifest, func() map[string]network.Channel {
		return map[string]network.Channel{
			remote.Node().String(): remote.Channel(),
		}
	})

	ss.WhenFinished(func(height base.Height) {
		finishedChan <- height
	})
	t.NoError(ss.Start())

	defer ss.Stop()

	for i := baseManifest.Height().Int64() + 1; i <= target.Int64(); i++ {
		isFinished, err := ss.Add(base.Height(i), []base.Node{remote.Node()})
		t.NoError(err)
		t.False(isFinished)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

end:
	for {
		select {
		case <-ctx.Done():
			t.NoError(errors.Errorf("timeout to wait to be finished"))

			break end
		case height := <-finishedChan:
			if height == target {
				break end
			}
		}
	}
}

func (t *testSyncers) TestMangledFinishedOrder() {
	ls := t.Locals(2)
	local, remote := ls[0], ls[1]

	t.SetupNodes(local, []*Local{remote})

	target := t.LastManifest(local.Database()).Height() + 10
	t.GenerateBlocks([]*Local{remote}, target)

	baseManifest, found, err := local.Database().LastManifest()
	t.NoError(err)
	t.True(found)

	finishedChan := make(chan base.Height, 10)

	ss := NewSyncers(local.Database(), local.Blockdata(), local.Policy(), baseManifest, func() map[string]network.Channel {
		return map[string]network.Channel{
			remote.Node().String(): remote.Channel(),
		}
	})

	ss.WhenFinished(func(height base.Height) {
		finishedChan <- height
	})
	t.NoError(ss.Start())

	defer ss.Stop()

	isFinished, err := ss.Add(target-1, []base.Node{remote.Node()})
	t.NoError(err)
	t.False(isFinished)
	isFinished, err = ss.Add(target, []base.Node{remote.Node()})
	t.NoError(err)
	t.False(isFinished)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

end:
	for {
		select {
		case <-ctx.Done():
			t.NoError(errors.Errorf("timeout to wait to be finished"))

			break end
		case height := <-finishedChan:
			if target == height {
				break end
			}
		}
	}
}

func (t *testSyncers) TestAddAfterFinished() {
	ls := t.Locals(2)
	local, remote := ls[0], ls[1]

	t.SetupNodes(local, []*Local{remote})

	target := t.LastManifest(local.Database()).Height() + 10
	t.GenerateBlocks([]*Local{remote}, target)

	baseManifest, found, err := local.Database().LastManifest()
	t.NoError(err)
	t.True(found)

	ss := NewSyncers(local.Database(), local.Blockdata(), local.Policy(), baseManifest, func() map[string]network.Channel {
		return map[string]network.Channel{
			remote.Node().String(): remote.Channel(),
		}
	})

	finishedChan := make(chan base.Height, 10)
	ss.WhenFinished(func(height base.Height) {
		finishedChan <- height
	})

	t.NoError(ss.Start())

	defer ss.Stop()

	isFinished, err := ss.Add(target-3, []base.Node{remote.Node()})
	t.NoError(err)
	t.False(isFinished)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

end0:
	for {
		select {
		case <-ctx.Done():
			t.NoError(errors.Errorf("timeout to wait to be finished"))

			break end0
		case height := <-finishedChan:
			if target-3 == height {
				break end0
			}
		}
	}

	isFinished, err = ss.Add(target, []base.Node{remote.Node()})
	t.NoError(err)
	t.False(isFinished)

	ctx, cancel = context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

end1:
	for {
		select {
		case <-ctx.Done():
			t.NoError(errors.Errorf("timeout to wait to be finished"))

			break end1
		case height := <-finishedChan:
			if target == height {
				break end1
			}
		}
	}
}

func (t *testSyncers) TestStopNotFinished() {
	ls := t.Locals(2)
	local, remote := ls[0], ls[1]

	t.SetupNodes(local, []*Local{remote})

	target := t.LastManifest(local.Database()).Height() + 10
	t.GenerateBlocks([]*Local{remote}, target)

	baseManifest, found, err := local.Database().LastManifest()
	t.NoError(err)
	t.True(found)

	ss := NewSyncers(local.Database(), local.Blockdata(), local.Policy(), baseManifest, func() map[string]network.Channel {
		return map[string]network.Channel{
			remote.Node().String(): remote.Channel(),
		}
	})
	ss.limitBlocksPerSyncer = 1

	t.NoError(ss.Start())

	isFinished, err := ss.Add(target, []base.Node{remote.Node()})
	t.NoError(err)
	t.False(isFinished)

	<-time.After(time.Millisecond * 100)

	t.NoError(ss.Stop())
	t.Nil(ss.LastSyncer())

	<-time.After(time.Second * 4)
}

func TestSyncers(t *testing.T) {
	suite.Run(t, new(testSyncers))
}
