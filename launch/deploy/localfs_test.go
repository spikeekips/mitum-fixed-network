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

type testBlockdataCleaner struct {
	isaac.BaseTest
	local *isaac.Local
}

func (t *testBlockdataCleaner) SetupTest() {
	t.BaseTest.SetupTest()
	t.local = t.Locals(1)[0]
}

func (t *testBlockdataCleaner) TestNew() {
	m, found, err := t.local.Database().LastManifest()
	t.NoError(err)
	t.True(found)

	lbd := t.local.Blockdata().(*localfs.Blockdata)

	bc := NewBlockdataCleaner(lbd, time.Millisecond)
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

func (t *testBlockdataCleaner) TestUnknownHeight() {
	m, found, err := t.local.Database().LastManifest()
	t.NoError(err)
	t.True(found)

	lbd := t.local.Blockdata().(*localfs.Blockdata)

	bc := NewBlockdataCleaner(lbd, time.Millisecond)

	err = bc.Add(m.Height() + 1)
	t.True(errors.Is(err, util.NotFoundError))
}

func (t *testBlockdataCleaner) TestAddRemovedHeights() {
	m, found, err := t.local.Database().LastManifest()
	t.NoError(err)
	t.True(found)

	lbd := t.local.Blockdata().(*localfs.Blockdata)
	t.NoError(lbd.Remove(m.Height()))
	t.NoError(lbd.Remove(m.Height() - 1))

	bc := NewBlockdataCleaner(lbd, time.Millisecond)
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

func (t *testBlockdataCleaner) TestAddRemovedHeightsWithStart() {
	m, found, err := t.local.Database().LastManifest()
	t.NoError(err)
	t.True(found)

	lbd := t.local.Blockdata().(*localfs.Blockdata)
	t.NoError(lbd.Remove(m.Height()))

	bc := NewBlockdataCleaner(lbd, time.Millisecond)
	bc.interval = time.Second
	t.NoError(bc.Start())
	defer bc.Stop()

	<-time.After(time.Second * 2)

	targets := bc.currentTargets()
	t.Equal(0, len(targets))
}

func TestBlockdataCleaner(t *testing.T) {
	suite.Run(t, new(testBlockdataCleaner))
}
