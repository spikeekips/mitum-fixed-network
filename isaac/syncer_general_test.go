package isaac

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	channetwork "github.com/spikeekips/mitum/network/gochan"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
)

type testGeneralSyncer struct {
	baseTestStateHandler

	sf base.Suffrage
}

func (t *testGeneralSyncer) setup(local *Localstate, others []*Localstate) {
	var nodes []*Localstate = []*Localstate{local}
	nodes = append(nodes, others...)

	lastHeight := t.lastManifest(local.Storage()).Height()

	for _, l := range nodes {
		t.NoError(l.Storage().Clean())
	}

	bg, err := NewDummyBlocksV0Generator(
		local,
		lastHeight,
		t.suffrage(local, nodes...),
		nodes,
	)
	t.NoError(err)
	t.NoError(bg.Generate(true))

	for _, st := range nodes {
		nch := st.Node().Channel().(*channetwork.NetworkChanChannel)
		nch.SetGetManifests(func(heights []base.Height) ([]block.Manifest, error) {
			var bs []block.Manifest
			for _, h := range heights {
				m, err := st.Storage().ManifestByHeight(h)
				if err != nil {
					if xerrors.Is(err, storage.NotFoundError) {
						break
					}

					return nil, err
				}

				bs = append(bs, m)
			}

			return bs, nil
		})

		nch.SetGetBlocks(func(heights []base.Height) ([]block.Block, error) {
			var bs []block.Block
			for _, h := range heights {
				m, err := st.Storage().BlockByHeight(h)
				if err != nil {
					if xerrors.Is(err, storage.NotFoundError) {
						break
					}

					return nil, err
				}

				bs = append(bs, m)
			}

			return bs, nil
		})
	}
}

func (t *testGeneralSyncer) generateBlocks(localstates []*Localstate, targetHeight base.Height) {
	bg, err := NewDummyBlocksV0Generator(
		localstates[0],
		targetHeight,
		t.suffrage(localstates[0], localstates...),
		localstates,
	)
	t.NoError(err)
	t.NoError(bg.Generate(false))
}

func (t *testGeneralSyncer) emptyLocalstate() *Localstate {
	lst := t.Storage(nil, nil)
	localNode := RandomLocalNode(util.UUID().String(), nil)
	localstate, err := NewLocalstate(lst, localNode, TestNetworkID)
	t.NoError(err)

	return localstate
}

func (t *testGeneralSyncer) TestInvalidFrom() {
	base := t.lastManifest(t.localstate.Storage()).Height()
	{ // lower than base
		_, err := NewGeneralSyncer(t.localstate, []Node{t.remoteState.Node()}, base-1, base+2)
		t.Contains(err.Error(), "lower than last block")
	}

	{ // same with base
		_, err := NewGeneralSyncer(t.localstate, []Node{t.remoteState.Node()}, base, base+2)
		t.Contains(err.Error(), "same or lower than last block")
	}

	{ // higher than to
		_, err := NewGeneralSyncer(t.localstate, []Node{t.remoteState.Node()}, base+3, base+2)
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
		_, err := NewGeneralSyncer(localstate, []Node{localstate.Node()}, base+1, base+2)
		t.Contains(err.Error(), "same with local node")
	}
}

func (t *testGeneralSyncer) TestNew() {
	ls := t.localstates(2)
	localstate, remoteState := ls[0], ls[1]

	t.setup(localstate, []*Localstate{remoteState})

	target := t.lastManifest(localstate.Storage()).Height() + 1
	t.generateBlocks([]*Localstate{remoteState}, target)

	cs, err := NewGeneralSyncer(localstate, []Node{remoteState.Node()}, target, target)
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

	cs, err := NewGeneralSyncer(localstate, []Node{rn0.Node(), rn1.Node(), rn2.Node()}, base+1, target)
	t.NoError(err)

	defer cs.Close()

	cs.reset()
	t.NoError(cs.headAndTailManifests())

	{
		b, err := cs.storage().Manifest(base + 1)
		t.NoError(err)
		t.Equal(base+1, b.Height())
	}

	{
		b, err := cs.storage().Manifest(target)
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

	cs, err := NewGeneralSyncer(localstate, []Node{rn0.Node(), rn1.Node(), rn2.Node()}, baseBlock.Height()+1, target)
	t.NoError(err)

	defer cs.Close()

	cs.reset()
	cs.setBaseManifest(baseBlock)
	t.NoError(cs.prepare())

	for i := baseBlock.Height().Int64() + 1; i < target.Int64()+1; i++ {
		b, err := cs.storage().Manifest(base.Height(i))
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

	cs, err := NewGeneralSyncer(localstate, []Node{rn0.Node(), rn1.Node(), rn2.Node()}, baseHeight+1, target)
	t.NoError(err)

	defer cs.Close()

	cs.reset()
	t.NoError(cs.headAndTailManifests())
	t.NoError(cs.fillManifests())
	t.NoError(cs.startBlocks())

	for i := baseHeight.Int64() + 1; i < target.Int64()+1; i++ {
		b, err := cs.storage().Manifest(base.Height(i))
		t.NoError(err)

		t.Equal(i, b.Height().Int64())
	}

	for i := baseHeight.Int64() + 1; i < target.Int64()+1; i++ {
		b, err := cs.storage().Block(base.Height(i))
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

	cs, err := NewGeneralSyncer(localstate, []Node{rn0.Node(), rn1.Node(), rn2.Node()}, baseHeight+1, target)
	t.NoError(err)

	defer cs.Close()

	cs.reset()
	t.NoError(cs.prepare())

	for i := baseHeight.Int64() + 1; i < target.Int64()+1; i++ {
		b, err := cs.storage().Manifest(base.Height(i))
		t.NoError(err)

		t.Equal(i, b.Height().Int64())
	}

	t.Equal(SyncerPrepared, cs.State())

	t.NoError(cs.startBlocks())

	for i := baseHeight.Int64() + 1; i < target.Int64()+1; i++ {
		b, err := cs.storage().Block(base.Height(i))
		t.NoError(err)
		t.Equal(b.Height(), base.Height(i))

		_, err = localstate.Storage().BlockByHeight(base.Height(i))
		t.True(xerrors.Is(err, storage.NotFoundError))
	}

	t.NoError(cs.commit())

	for i := baseHeight.Int64() + 1; i < target.Int64()+1; i++ {
		b, err := localstate.Storage().BlockByHeight(base.Height(i))
		t.NoError(err)
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

	cs, err := NewGeneralSyncer(localstate, []Node{rn0.Node(), rn1.Node(), rn2.Node()}, baseBlock.Height()+1, target)
	t.NoError(err)

	defer cs.Close()

	stateChan := make(chan Syncer)
	finishedChan := make(chan Syncer)

	go func() {
	end:
		for {
			select {
			case ss := <-stateChan:
				if ss.State() != SyncerSaved {
					continue
				}

				finishedChan <- ss
				break end
			}
		}
	}()

	cs.SetStateChan(stateChan)

	t.NoError(cs.Prepare(baseBlock))
	t.NoError(cs.Save())

	select {
	case <-time.After(time.Second * 5):
		t.NoError(xerrors.Errorf("timeout to wait to be finished"))
	case ss := <-finishedChan:
		t.Equal(SyncerSaved, ss.State())
		t.Equal(baseBlock.Height()+1, ss.HeightFrom())
		t.Equal(target, ss.HeightTo())
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

	cs, err := NewGeneralSyncer(syncNode, []Node{localstate.Node()}, base.PreGenesisHeight, target.Height())
	t.NoError(err)

	defer cs.Close()

	stateChan := make(chan Syncer)
	finishedChan := make(chan Syncer)

	go func() {
	end:
		for {
			select {
			case ss := <-stateChan:
				if ss.State() != SyncerSaved {
					continue
				}

				finishedChan <- ss
				break end
			}
		}
	}()

	cs.SetStateChan(stateChan)

	t.NoError(cs.Prepare(nil))
	t.NoError(cs.Save())

	select {
	case <-time.After(time.Second * 5):
		t.NoError(xerrors.Errorf("timeout to wait to be finished"))
	case ss := <-finishedChan:
		t.Equal(SyncerSaved, ss.State())
		t.Equal(base.PreGenesisHeight, ss.HeightFrom())
		t.Equal(target.Height(), ss.HeightTo())
	}
}

func (t *testGeneralSyncer) TestSyncingHandlerFromBallot() {
	ls := t.localstates(4)
	localstate, rn0, rn1, rn2 := ls[0], ls[1], ls[2], ls[3]

	t.setup(localstate, []*Localstate{rn0, rn1, rn2})

	baseBlock := t.lastManifest(localstate.Storage())
	target := baseBlock.Height() + 5
	t.generateBlocks([]*Localstate{rn0, rn1, rn2}, target)

	cs, err := NewStateSyncingHandler(localstate, nil)
	t.NoError(err)

	blt := t.newINITBallot(rn0, base.Round(0), nil)

	t.NoError(cs.Activate(NewStateChangeContext(base.StateJoining, base.StateSyncing, nil, blt)))

	finishedChan := make(chan struct{})
	go func() {
		for {
			b, err := localstate.Storage().LastManifest()
			t.NoError(err)
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

	cs, err := NewStateSyncingHandler(localstate, nil)
	t.NoError(err)

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

	cs, err := NewStateSyncingHandler(localstate, nil)
	t.NoError(err)

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
	ch.SetGetManifests(func(heights []base.Height) ([]block.Manifest, error) {
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

	cs, err := NewGeneralSyncer(localstate, []Node{rn0.Node()}, baseBlock.Height()+1, target)
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
	ch.SetGetManifests(func(heights []base.Height) ([]block.Manifest, error) {
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

	cs, err := NewGeneralSyncer(localstate, []Node{rn0.Node()}, baseBlock.Height()+1, target)
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
	ch.SetGetManifests(func(heights []base.Height) ([]block.Manifest, error) {
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

	cs, err := NewGeneralSyncer(localstate, []Node{rn0.Node()}, baseBlock.Height()+1, target)
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
	ch.SetGetBlocks(func(heights []base.Height) ([]block.Block, error) {
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

	cs, err := NewGeneralSyncer(localstate, []Node{rn0.Node()}, baseBlock.Height()+1, target)
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

func TestGeneralSyncer(t *testing.T) {
	suite.Run(t, new(testGeneralSyncer))
}
