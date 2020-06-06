// +build mongodb

package isaac

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/network"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
)

func (t *testSyncers) TestSaveLastBlock() {
	if t.DBType != "mongodb" {
		return
	}

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

	orig := localstate.Storage().(DummyMongodbStorage)

	st, err := mongodbstorage.NewStorage(orig.Client(), t.Encs, t.BSONEnc)
	t.NoError(err)
	d := DummyMongodbStorage{st}

	dlm, found, err := d.LastManifest()
	t.NoError(err)
	t.True(found)

	t.compareManifest(rm, dlm)
}

func TestSyncersMongodb(t *testing.T) {
	handler := new(testSyncers)
	handler.DBType = "mongodb"

	suite.Run(t, handler)
}
