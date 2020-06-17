package isaac

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/network"
)

type testSyncers struct {
	baseTestSyncer
}

func (t *testSyncers) TestNew() {
	ls := t.localstates(2)
	localstate, remoteState := ls[0], ls[1]

	t.setup(localstate, []*Localstate{remoteState})

	target := t.lastManifest(localstate.Storage()).Height() + 2
	t.generateBlocks([]*Localstate{remoteState}, target)

	baseManifest, found, err := localstate.Storage().LastManifest()
	t.NoError(err)
	t.True(found)

	finishedChan := make(chan base.Height)

	ss := NewSyncers(localstate, baseManifest)
	ss.WhenFinished(func(height base.Height) {
		finishedChan <- height
	})
	t.NoError(ss.Start())

	defer ss.Stop()

	t.NoError(ss.Add(target, []network.Node{remoteState.Node()}))

	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("timeout to wait to be finished"))
	case height := <-finishedChan:
		t.Equal(target, height)
		break
	}

	rm, found, err := remoteState.Storage().LastManifest()
	t.NoError(err)
	t.True(found)
	lm, found, err := localstate.Storage().LastManifest()
	t.NoError(err)
	t.True(found)

	t.compareManifest(rm, lm)
}

func (t *testSyncers) TestMultipleSyncers() {
	ls := t.localstates(2)
	localstate, remoteState := ls[0], ls[1]

	t.setup(localstate, []*Localstate{remoteState})

	target := t.lastManifest(localstate.Storage()).Height() + 2
	t.generateBlocks([]*Localstate{remoteState}, target)

	baseManifest, found, err := localstate.Storage().LastManifest()
	t.NoError(err)
	t.True(found)

	finishedChan := make(chan base.Height)

	ss := NewSyncers(localstate, baseManifest)
	ss.WhenFinished(func(height base.Height) {
		finishedChan <- height
	})
	t.NoError(ss.Start())

	defer ss.Stop()

	for i := baseManifest.Height().Int64() + 1; i <= target.Int64(); i++ {
		t.NoError(ss.Add(base.Height(i), []network.Node{remoteState.Node()}))
	}

	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("timeout to wait to be finished"))
	case height := <-finishedChan:
		t.Equal(target, height)
		break
	}
}

func (t *testSyncers) TestMangledFinishedOrder() {
	ls := t.localstates(2)
	localstate, remoteState := ls[0], ls[1]

	t.setup(localstate, []*Localstate{remoteState})

	target := t.lastManifest(localstate.Storage()).Height() + 10
	t.generateBlocks([]*Localstate{remoteState}, target)

	baseManifest, found, err := localstate.Storage().LastManifest()
	t.NoError(err)
	t.True(found)

	finishedChan := make(chan base.Height)

	ss := NewSyncers(localstate, baseManifest)

	ss.WhenFinished(func(height base.Height) {
		finishedChan <- height
	})
	t.NoError(ss.Start())

	defer ss.Stop()

	t.NoError(ss.Add(target-1, []network.Node{remoteState.Node()}))
	t.NoError(ss.Add(target, []network.Node{remoteState.Node()}))

	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("timeout to wait to be finished"))
	case height := <-finishedChan:
		t.Equal(target, height)
		break
	}
}

func (t *testSyncers) TestAddAfterFinished() {
	ls := t.localstates(2)
	localstate, remoteState := ls[0], ls[1]

	t.setup(localstate, []*Localstate{remoteState})

	target := t.lastManifest(localstate.Storage()).Height() + 10
	t.generateBlocks([]*Localstate{remoteState}, target)

	baseManifest, found, err := localstate.Storage().LastManifest()
	t.NoError(err)
	t.True(found)

	ss := NewSyncers(localstate, baseManifest)

	finishedChan := make(chan base.Height)
	ss.WhenFinished(func(height base.Height) {
		finishedChan <- height
	})
	t.NoError(ss.Start())

	defer ss.Stop()

	t.NoError(ss.Add(target-3, []network.Node{remoteState.Node()}))

	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("timeout to wait to be finished"))
	case height := <-finishedChan:
		t.Equal(target-3, height)
		break
	}

	t.NoError(ss.Add(target, []network.Node{remoteState.Node()}))

	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("timeout to wait to be finished"))
	case height := <-finishedChan:
		t.Equal(target, height)
		break
	}
}

func TestSyncers(t *testing.T) {
	suite.Run(t, new(testSyncers))
}
