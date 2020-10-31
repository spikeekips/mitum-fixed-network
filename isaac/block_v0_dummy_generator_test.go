package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/storage"
)

type testBlockV0DummyGenerator struct {
	baseTestStateHandler
}

func (t *testBlockV0DummyGenerator) TestCreate() {
	all := t.locals(3)

	for _, l := range all {
		t.NoError(l.Storage().Clean())
	}

	defer t.closeStates(all...)

	var suffrage base.Suffrage
	{
		nodes := make([]base.Address, len(all))
		for i := range all {
			nodes[i] = all[i].Node().Address()
		}

		suffrage = t.suffrage(all[0], all...)
	}

	lastHeight := base.Height(3)
	bg, err := NewDummyBlocksV0Generator(all[0], lastHeight, suffrage, all)
	t.NoError(err)

	t.NoError(bg.Generate(true))

	for i := int64(0); i < lastHeight.Int64(); i++ {
		hashes := map[string]struct{}{}
		for nodeid, l := range all {
			blk, err := l.BlockFS().Load(base.Height(i))
			t.NoError(err)

			t.NoError(err, "node=%d height=%d", nodeid, i)
			t.NotNil(blk, "node=%d height=%d", nodeid, i)
			t.NoError(blk.IsValid(all[0].Policy().NetworkID()), "height=%d", i)

			hashes[blk.Hash().String()] = struct{}{}
		}

		t.Equal(1, len(hashes), "check block hashes are matched")
	}
}

func (t *testBlockV0DummyGenerator) TestCleanByHeight() {
	local := t.locals(1)[0]

	lastManifest, _, _ := local.Storage().LastManifest()

	h, err := local.BlockFS().Exists(lastManifest.Height())
	t.NoError(err)
	t.NotNil(h)

	t.NoError(local.Storage().CleanByHeight(lastManifest.Height()))
	t.NoError(local.BlockFS().CleanByHeight(lastManifest.Height()))

	l, _, _ := local.Storage().LastManifest()
	t.Equal(lastManifest.Height()-1, l.Height())

	h, err = local.BlockFS().Exists(lastManifest.Height())
	t.True(xerrors.Is(err, storage.NotFoundError))
	t.Nil(h)
}

func TestBlockV0DummyGenerator(t *testing.T) {
	suite.Run(t, new(testBlockV0DummyGenerator))
}
