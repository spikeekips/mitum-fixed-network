package isaac

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
)

type testGeneralSyncer struct {
	sync.Mutex
	baseTestStateHandler

	sf Suffrage
}

func TestGeneralSyncer(t *testing.T) {
	suite.Run(t, new(testGeneralSyncer))
}

func (t *testGeneralSyncer) setup(local *Localstate, localstates []*Localstate) {
	var nodes []*Localstate = []*Localstate{local}
	nodes = append(nodes, localstates...)

	bg, err := NewDummyBlocksV0Generator(
		local,
		local.LastBlock().Height(),
		t.suffrage(local, nodes...),
		nodes,
	)
	t.NoError(err)
	t.NoError(bg.Generate(true))

	t.Lock()
	defer t.Unlock()

	for _, st := range nodes {
		nch := st.Node().Channel().(*NetworkChanChannel)
		nch.SetGetBlockManifests(func(heights []Height) ([]BlockManifest, error) {
			var bs []BlockManifest
			for _, h := range heights {
				m, err := st.Storage().BlockManifestByHeight(h)
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

		nch.SetGetBlocks(func(heights []Height) ([]Block, error) {
			var bs []Block
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

func (t *testGeneralSyncer) generateBlocks(localstates []*Localstate, targetHeight Height) {
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
	lst := NewMemStorage(t.encs, t.enc)
	localNode := RandomLocalNode(util.UUID().String(), nil)
	localstate, err := NewLocalstate(lst, localNode, TestNetworkID)
	t.NoError(err)

	return localstate
}

func (t *testGeneralSyncer) TestInvalidFrom() {
	base := t.localstate.LastBlock().Height()
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
	localstate, _ := t.states()
	t.setup(localstate, nil)

	base := localstate.LastBlock().Height()

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
	localstate, remoteState := t.states()
	t.setup(localstate, []*Localstate{remoteState})

	target := localstate.LastBlock().Height() + 1
	t.generateBlocks([]*Localstate{remoteState}, target)

	cs, err := NewGeneralSyncer(localstate, []Node{remoteState.Node()}, target, target)
	t.NoError(err)

	_ = (interface{})(cs).(Syncer)
	t.Implements((*Syncer)(nil), cs)

	t.Equal(SyncerCreated, cs.State())
}

// TestHeadAndTailManifests setups 4 nodes and 3 nodes has higher blocks rather
// than 1 node.
func (t *testGeneralSyncer) TestHeadAndTailManifests() {
	localstate, rn0 := t.states()
	rn1, rn2 := t.states()
	t.setup(localstate, []*Localstate{rn0, rn1, rn2})

	base := localstate.LastBlock().Height()
	target := base + 5
	t.generateBlocks([]*Localstate{rn0, rn1, rn2}, target)

	cs, err := NewGeneralSyncer(localstate, []Node{rn0.Node(), rn1.Node(), rn2.Node()}, base+1, target)
	t.NoError(err)

	cs.reset()
	t.NoError(cs.headAndTailManifests())

	{
		b, err := cs.storage.Manifest(base + 1)
		t.NoError(err)
		t.Equal(base+1, b.Height())
	}

	{
		b, err := cs.storage.Manifest(target)
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
	localstate, rn0 := t.states()
	rn1, rn2 := t.states()
	t.setup(localstate, []*Localstate{rn0, rn1, rn2})

	base := localstate.LastBlock()
	target := base.Height() + 5
	t.generateBlocks([]*Localstate{rn0, rn1, rn2}, target)

	cs, err := NewGeneralSyncer(localstate, []Node{rn0.Node(), rn1.Node(), rn2.Node()}, base.Height()+1, target)
	t.NoError(err)

	cs.reset()
	cs.baseManifest = base
	t.NoError(cs.prepare())

	for i := base.Height().Int64() + 1; i < target.Int64()+1; i++ {
		b, err := cs.storage.Manifest(Height(i))
		t.NoError(err)

		t.Equal(i, b.Height().Int64())
	}
}

// TestFetchBlocks setups 4 nodes and 3 nodes has higher blocks rather
// than 1 node.
func (t *testGeneralSyncer) TestFetchBlocks() {
	localstate, rn0 := t.states()
	rn1, rn2 := t.states()
	t.setup(localstate, []*Localstate{rn0, rn1, rn2})

	base := localstate.LastBlock().Height()
	target := base + 5
	t.generateBlocks([]*Localstate{rn0, rn1, rn2}, target)

	cs, err := NewGeneralSyncer(localstate, []Node{rn0.Node(), rn1.Node(), rn2.Node()}, base+1, target)
	t.NoError(err)

	cs.reset()
	t.NoError(cs.headAndTailManifests())
	t.NoError(cs.fillManifests())
	t.NoError(cs.startBlocks())

	for i := base.Int64() + 1; i < target.Int64()+1; i++ {
		b, err := cs.storage.Manifest(Height(i))
		t.NoError(err)

		t.Equal(i, b.Height().Int64())
	}

	for i := base.Int64() + 1; i < target.Int64()+1; i++ {
		b, err := cs.storage.Block(Height(i))
		t.NoError(err)
		t.Equal(b.Height(), Height(i))
	}
}

func (t *testGeneralSyncer) TestSaveBlocks() {
	localstate, rn0 := t.states()
	rn1, rn2 := t.states()
	t.setup(localstate, []*Localstate{rn0, rn1, rn2})

	base := localstate.LastBlock().Height()
	target := base + 5
	t.generateBlocks([]*Localstate{rn0, rn1, rn2}, target)

	cs, err := NewGeneralSyncer(localstate, []Node{rn0.Node(), rn1.Node(), rn2.Node()}, base+1, target)
	t.NoError(err)

	cs.reset()
	t.NoError(cs.prepare())

	for i := base.Int64() + 1; i < target.Int64()+1; i++ {
		b, err := cs.storage.Manifest(Height(i))
		t.NoError(err)

		t.Equal(i, b.Height().Int64())
	}

	t.Equal(SyncerPrepared, cs.State())

	t.NoError(cs.startBlocks())

	for i := base.Int64() + 1; i < target.Int64()+1; i++ {
		b, err := cs.storage.Block(Height(i))
		t.NoError(err)
		t.Equal(b.Height(), Height(i))

		_, err = localstate.Storage().BlockByHeight(Height(i))
		t.True(xerrors.Is(err, storage.NotFoundError))
	}

	t.NoError(cs.commit())

	for i := base.Int64() + 1; i < target.Int64()+1; i++ {
		b, err := localstate.Storage().BlockByHeight(Height(i))
		t.NoError(err)
		t.Equal(b.Height(), Height(i))
	}
}

func (t *testGeneralSyncer) TestFinishedChan() {
	localstate, rn0 := t.states()
	rn1, rn2 := t.states()
	t.setup(localstate, []*Localstate{rn0, rn1, rn2})

	base := localstate.LastBlock()
	target := base.Height() + 5
	t.generateBlocks([]*Localstate{rn0, rn1, rn2}, target)

	cs, err := NewGeneralSyncer(localstate, []Node{rn0.Node(), rn1.Node(), rn2.Node()}, base.Height()+1, target)
	t.NoError(err)

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

	t.NoError(cs.Prepare(base))
	t.NoError(cs.Save())

	select {
	case <-time.After(time.Second * 5):
		t.NoError(xerrors.Errorf("timeout to wait to be finished"))
	case ss := <-finishedChan:
		t.Equal(SyncerSaved, ss.State())
		t.Equal(base.Height()+1, ss.HeightFrom())
		t.Equal(target, ss.HeightTo())
	}
}

func (t *testGeneralSyncer) TestFromGenesis() {
	localstate, _ := t.states()
	t.setup(localstate, nil)

	syncNode := t.emptyLocalstate()
	t.NoError(localstate.Nodes().Add(syncNode.Node()))

	target := localstate.LastBlock()

	cs, err := NewGeneralSyncer(syncNode, []Node{localstate.Node()}, 0, target.Height())
	t.NoError(err)

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
		t.Equal(Height(0), ss.HeightFrom())
		t.Equal(target.Height(), ss.HeightTo())
	}
}

func (t *testGeneralSyncer) TestSyncingHandlerFromBallot() {
	localstate, rn0 := t.states()
	rn1, rn2 := t.states()
	t.setup(localstate, []*Localstate{rn0, rn1, rn2})

	base := localstate.LastBlock()
	target := base.Height() + 5
	t.generateBlocks([]*Localstate{rn0, rn1, rn2}, target)

	cs, err := NewStateSyncingHandler(localstate, nil)
	t.NoError(err)

	var ballot Ballot
	{
		b := t.newINITBallot(rn0, Round(0))

		vp, err := t.newVoteproof(b.Stage(), b.INITBallotFactV0, rn0, rn1, rn2)
		t.NoError(err)
		b.voteproof = vp

		t.NoError(b.Sign(rn0.Node().Privatekey(), nil))

		ballot = b
	}

	t.NoError(cs.Activate(NewStateChangeContext(StateJoining, StateSyncing, nil, ballot)))

	finishedChan := make(chan struct{})
	go func() {
		for {
			b, err := localstate.Storage().LastBlock()
			t.NoError(err)
			if b.Height() == ballot.Height()-1 {
				finishedChan <- struct{}{}
				break
			}

			<-time.After(time.Millisecond * 10)
		}
	}()

	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("timeout to wait to be finished"))
		break
	case <-finishedChan:
		break
	}
}

func (t *testGeneralSyncer) TestSyncingHandlerFromINITVoteproof() {
	localstate, rn0 := t.states()
	rn1, rn2 := t.states()
	t.setup(localstate, []*Localstate{rn0, rn1, rn2})

	t.NoError(localstate.Nodes().Add(rn1.Node()))
	t.NoError(localstate.Nodes().Add(rn2.Node()))

	base := localstate.LastBlock()
	target := base.Height() + 5
	t.generateBlocks([]*Localstate{rn0, rn1, rn2}, target)

	cs, err := NewStateSyncingHandler(localstate, nil)
	t.NoError(err)

	var voteproof Voteproof
	{
		b := t.newINITBallot(rn0, Round(0))

		vp, err := t.newVoteproof(b.Stage(), b.INITBallotFactV0, rn0, rn1, rn2)
		t.NoError(err)

		voteproof = vp
	}

	t.NoError(cs.Activate(NewStateChangeContext(StateJoining, StateSyncing, voteproof, nil)))

	stopChan := make(chan struct{})
	finishedChan := make(chan struct{})
	go func() {
	end:
		for {
			select {
			case <-stopChan:
				break end
			default:
				if localstate.LastBlock().Height() == voteproof.Height()-1 {
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
	localstate, rn0 := t.states()
	rn1, rn2 := t.states()
	t.setup(localstate, []*Localstate{rn0, rn1, rn2})

	t.NoError(localstate.Nodes().Add(rn1.Node()))
	t.NoError(localstate.Nodes().Add(rn2.Node()))

	base := localstate.LastBlock()
	target := base.Height() + 5
	t.generateBlocks([]*Localstate{rn0, rn1, rn2}, target)

	cs, err := NewStateSyncingHandler(localstate, nil)
	t.NoError(err)

	var voteproof Voteproof
	{
		ab, err := NewACCEPTBallotV0FromLocalstate(rn0, Round(0), rn0.LastBlock())
		ab.height = rn0.LastBlock().Height()
		vp, err := t.newVoteproof(ab.Stage(), ab.ACCEPTBallotFactV0, rn0, rn1, rn2)
		t.NoError(err)

		voteproof = vp
	}

	t.NoError(cs.Activate(NewStateChangeContext(StateJoining, StateSyncing, voteproof, nil)))

	stopChan := make(chan struct{})
	finishedChan := make(chan struct{})
	go func() {
	end:
		for {
			select {
			case <-stopChan:
				break end
			default:
				if localstate.LastBlock().Height() == voteproof.Height() {
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
	localstate, rn0 := t.states()
	t.setup(localstate, []*Localstate{rn0})

	base := localstate.LastBlock()
	target := base.Height() + 5
	t.generateBlocks([]*Localstate{rn0}, target)

	head := base.Height() + 1
	ch := rn0.Node().Channel().(*NetworkChanChannel)
	orig := ch.getBlockManifests
	ch.SetGetBlockManifests(func(heights []Height) ([]BlockManifest, error) {
		var bs []BlockManifest
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

	cs, err := NewGeneralSyncer(localstate, []Node{rn0.Node()}, base.Height()+1, target)
	t.NoError(err)

	cs.reset()

	err = cs.headAndTailManifests()
	t.Error(err)
}

func (t *testGeneralSyncer) TestMissingTail() {
	localstate, rn0 := t.states()
	t.setup(localstate, []*Localstate{rn0})

	base := localstate.LastBlock()
	target := base.Height() + 5
	t.generateBlocks([]*Localstate{rn0}, target)

	tail := target
	ch := rn0.Node().Channel().(*NetworkChanChannel)
	orig := ch.getBlockManifests
	ch.SetGetBlockManifests(func(heights []Height) ([]BlockManifest, error) {
		var bs []BlockManifest
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

	cs, err := NewGeneralSyncer(localstate, []Node{rn0.Node()}, base.Height()+1, target)
	t.NoError(err)

	cs.reset()

	err = cs.headAndTailManifests()
	t.Error(err)
}

func (t *testGeneralSyncer) TestMissingManifests() {
	localstate, rn0 := t.states()
	t.setup(localstate, []*Localstate{rn0})

	base := localstate.LastBlock()
	target := base.Height() + 5
	t.generateBlocks([]*Localstate{rn0}, target)

	missing := target - 1
	ch := rn0.Node().Channel().(*NetworkChanChannel)
	orig := ch.getBlockManifests
	ch.SetGetBlockManifests(func(heights []Height) ([]BlockManifest, error) {
		var bs []BlockManifest
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

	cs, err := NewGeneralSyncer(localstate, []Node{rn0.Node()}, base.Height()+1, target)
	t.NoError(err)

	cs.reset()

	err = cs.fillManifests()
	t.Error(err)
}

func (t *testGeneralSyncer) TestMissingBlocks() {
	localstate, rn0 := t.states()
	t.setup(localstate, []*Localstate{rn0})

	base := localstate.LastBlock()
	target := base.Height() + 5
	t.generateBlocks([]*Localstate{rn0}, target)

	missing := target - 1
	ch := rn0.Node().Channel().(*NetworkChanChannel)
	orig := ch.getBlocks
	ch.SetGetBlocks(func(heights []Height) ([]Block, error) {
		var bs []Block
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

	cs, err := NewGeneralSyncer(localstate, []Node{rn0.Node()}, base.Height()+1, target)
	t.NoError(err)

	cs.reset()

	t.NoError(cs.Prepare(base))

	err = cs.fetchBlocksByNodes()
	t.Error(err)
}
