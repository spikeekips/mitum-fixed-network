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

func (t *testBlockV0DummyGenerator) SetupTest() {
	t.baseTestStateHandler.SetupTest()
	baseLocalstate := t.baseTestStateHandler.localstate

	localstate, err := NewLocalstate(
		t.Storage(nil, nil),
		baseLocalstate.Node(),
		TestNetworkID,
	)
	t.NoError(err)
	t.localstate = localstate
}

func (t *testBlockV0DummyGenerator) localstates(n int) []*Localstate {
	ls := make([]*Localstate, n)

	var i int
	for {
		a, b := t.states()
		ls[i] = a
		i++
		if i == n {
			break
		}
		ls[i] = b
		i++
		if i == n {
			break
		}
	}

	return ls
}

func (t *testBlockV0DummyGenerator) TestCreate() {
	all := []*Localstate{t.localstate}
	all = append(all, t.localstates(2)...)

	defer t.closeStates(all...)

	var suffrage base.Suffrage
	{
		nodes := make([]base.Node, len(all))
		for i, o := range all {
			nodes[i] = o.Node()
		}

		suffrage = base.NewFixedSuffrage(t.localstate.Node(), nodes)
	}

	lastHeight := base.Height(10)
	bg, err := NewDummyBlocksV0Generator(t.localstate, lastHeight, suffrage, all)
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
