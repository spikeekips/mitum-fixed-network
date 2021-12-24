//go:build mongodb
// +build mongodb

package isaac

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/network"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/stretchr/testify/suite"
)

func (t *testSyncers) TestSaveLastBlock() {
	if t.DBType != "mongodb" {
		return
	}

	ls := t.Locals(2)
	local, remote := ls[0], ls[1]

	t.SetupNodes(local, []*Local{remote})

	target := t.LastManifest(local.Database()).Height() + 2
	t.GenerateBlocks([]*Local{remote}, target)

	baseManifest, found, err := local.Database().LastManifest()
	t.NoError(err)
	t.True(found)

	finishedChan := make(chan base.Height)

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

	isFinished, err := ss.Add(target, []base.Node{remote.Node()})
	t.NoError(err)
	t.False(isFinished)

	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("timeout to wait to be finished"))
	case height := <-finishedChan:
		t.Equal(target, height)
		break
	}

	rm, found, err := remote.Database().LastManifest()
	t.NoError(err)
	t.True(found)

	lm, found, err := local.Database().LastManifest()
	t.NoError(err)
	t.True(found)

	t.CompareManifest(rm, lm)

	orig := local.Database().(DummyMongodbDatabase)

	st, err := mongodbstorage.NewDatabase(orig.Client(), t.Encs, t.BSONEnc, cache.Dummy{})
	t.NoError(err)
	d := NewDummyMongodbDatabase(st)

	dlm, found, err := d.LastManifest()
	t.NoError(err)
	t.True(found)

	t.CompareManifest(rm, dlm)
}

func TestSyncersMongodb(t *testing.T) {
	handler := new(testSyncers)
	handler.DBType = "mongodb"

	suite.Run(t, handler)
}
