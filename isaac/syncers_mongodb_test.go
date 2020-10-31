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
	"github.com/spikeekips/mitum/util/cache"
)

func (t *testSyncers) TestSaveLastBlock() {
	if t.DBType != "mongodb" {
		return
	}

	ls := t.locals(2)
	local, remote := ls[0], ls[1]

	t.setup(local, []*Local{remote})

	target := t.lastManifest(local.Storage()).Height() + 2
	t.generateBlocks([]*Local{remote}, target)

	baseManifest, found, err := local.Storage().LastManifest()
	t.NoError(err)
	t.True(found)

	finishedChan := make(chan base.Height)

	ss := NewSyncers(local, baseManifest)
	ss.WhenFinished(func(height base.Height) {
		finishedChan <- height
	})
	t.NoError(ss.Start())

	defer ss.Stop()

	t.NoError(ss.Add(target, []network.Node{remote.Node()}))

	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("timeout to wait to be finished"))
	case height := <-finishedChan:
		t.Equal(target, height)
		break
	}

	rm, found, err := remote.Storage().LastManifest()
	t.NoError(err)
	t.True(found)

	lm, found, err := local.Storage().LastManifest()
	t.NoError(err)
	t.True(found)

	t.compareManifest(rm, lm)

	orig := local.Storage().(DummyMongodbStorage)

	st, err := mongodbstorage.NewStorage(orig.Client(), t.Encs, t.BSONEnc, cache.Dummy{})
	t.NoError(err)
	d := NewDummyMongodbStorage(st)

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
