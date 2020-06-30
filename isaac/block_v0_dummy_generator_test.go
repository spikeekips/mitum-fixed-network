package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
)

type testBlockV0DummyGenerator struct {
	baseTestStateHandler
}

func (t *testBlockV0DummyGenerator) TestCreate() {
	all := t.localstates(3)

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
			blk, found, err := l.Storage().BlockByHeight(base.Height(i))
			t.True(found)

			t.NoError(err, "node=%d height=%d", nodeid, i)
			t.NotNil(blk, "node=%d height=%d", nodeid, i)
			t.NoError(blk.IsValid(all[0].Policy().NetworkID()))

			hashes[blk.Hash().String()] = struct{}{}
		}

		t.Equal(1, len(hashes), "check block hashes are matched")
	}
}

func (t *testBlockV0DummyGenerator) TestCleanByHeight() {
	local := t.localstates(1)[0]

	lastManifest, _, _ := local.Storage().LastManifest()

	t.NoError(local.Storage().CleanByHeight(lastManifest.Height()))

	l, _, _ := local.Storage().LastManifest()
	t.Equal(lastManifest.Height()-1, l.Height())
}

func TestBlockV0DummyGenerator(t *testing.T) {
	suite.Run(t, new(testBlockV0DummyGenerator))
}
