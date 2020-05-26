package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/valuehash"
)

type testBlockV0DummyGenerator struct {
	baseTestStateHandler
	localstate *Localstate
}

func (t *testBlockV0DummyGenerator) TestCreate() {
	all := t.localstates(3)

	for _, l := range all {
		t.NoError(l.Storage().Clean())
	}

	defer t.closeStates(all...)

	var suffrage base.Suffrage
	{
		nodes := make([]base.Node, len(all))
		for i, o := range all {
			nodes[i] = o.Node()
		}

		suffrage = base.NewFixedSuffrage(all[0].Node(), nodes)
	}

	lastHeight := base.Height(10)
	bg, err := NewDummyBlocksV0Generator(all[0], lastHeight, suffrage, all)
	t.NoError(err)

	t.NoError(bg.Generate(true))

	for i := int64(0); i < lastHeight.Int64(); i++ {
		hashes := map[valuehash.Hash]struct{}{}
		for nodeid, l := range all {
			blk, err := l.Storage().BlockByHeight(base.Height(i))
			t.NoError(err, "node=%d height=%d", nodeid, i)
			t.NotNil(blk, "node=%d height=%d", nodeid, i)
			t.NoError(blk.IsValid(nil))

			hashes[blk.Hash()] = struct{}{}
		}

		t.Equal(1, len(hashes), "check block hashes are matched")
	}
}

func TestBlockV0DummyGenerator(t *testing.T) {
	suite.Run(t, new(testBlockV0DummyGenerator))
}
