package deploy

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/storage/blockdata/localfs"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/stretchr/testify/suite"
)

type testBlockDataCleaner struct {
	isaac.BaseTest
	local *isaac.Local
}

func (t *testBlockDataCleaner) SetupTest() {
	t.BaseTest.SetupTest()
	t.local = t.Locals(1)[0]
}

func (t *testBlockDataCleaner) TestNew() {
	m, found, err := t.local.Database().LastManifest()
	t.NoError(err)
	t.True(found)

	lbd := t.local.BlockData().(*localfs.BlockData)

	bc := NewBlockDataCleaner(lbd, time.Millisecond)
	bc.interval = time.Millisecond * 300

	t.NoError(bc.Add(m.Height()))

	t.NoError(bc.Start())
	defer bc.Stop()

	<-time.After(time.Second * 2)
	bc.removeAfter = time.Hour
	t.NoError(bc.Add(m.Height() - 1))

	<-time.After(time.Second * 1)

	remained := bc.currentTargets()
	t.T().Logf("remained: %v", remained)

	t.Equal(1, len(remained))
	_, found = remained[m.Height()-1]
	t.True(found)

	found, removed, err := lbd.ExistsReal(m.Height())
	t.NoError(err)
	t.False(found)
	t.False(removed)

	found, removed, err = lbd.ExistsReal(m.Height() - 1)
	t.NoError(err)
	t.True(found)
	t.True(removed)
}

func (t *testBlockDataCleaner) TestUnknownHeight() {
	m, found, err := t.local.Database().LastManifest()
	t.NoError(err)
	t.True(found)

	lbd := t.local.BlockData().(*localfs.BlockData)

	bc := NewBlockDataCleaner(lbd, time.Millisecond)

	err = bc.Add(m.Height() + 1)
	t.True(errors.Is(err, util.NotFoundError))
}

func (t *testBlockDataCleaner) TestAddRemovedHeights() {
	m, found, err := t.local.Database().LastManifest()
	t.NoError(err)
	t.True(found)

	lbd := t.local.BlockData().(*localfs.BlockData)
	t.NoError(lbd.Remove(m.Height()))
	t.NoError(lbd.Remove(m.Height() - 1))

	bc := NewBlockDataCleaner(lbd, time.Millisecond)
	t.NoError(bc.findRemoveds(context.Background()))

	targets := bc.currentTargets()
	t.Equal(2, len(targets))

	i, found := targets[m.Height()]
	t.True(found)
	t.True(localtime.UTCNow().After(i))

	i, found = targets[m.Height()-1]
	t.True(found)
	t.True(localtime.UTCNow().After(i))
}

func (t *testBlockDataCleaner) TestAddRemovedHeightsWithStart() {
	m, found, err := t.local.Database().LastManifest()
	t.NoError(err)
	t.True(found)

	lbd := t.local.BlockData().(*localfs.BlockData)
	t.NoError(lbd.Remove(m.Height()))

	bc := NewBlockDataCleaner(lbd, time.Millisecond)
	bc.interval = time.Second
	t.NoError(bc.Start())
	defer bc.Stop()

	<-time.After(time.Second * 2)

	targets := bc.currentTargets()
	t.Equal(0, len(targets))
}

func TestBlockDataCleaner(t *testing.T) {
	suite.Run(t, new(testBlockDataCleaner))
}
