package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/valuehash"
)

type testBlockV0DummyGenerator struct {
	baseTestStateHandler
	localstate *Localstate
}

func (t *testBlockV0DummyGenerator) SetupTest() {
	t.baseTestStateHandler.SetupTest()
	baseLocalstate := t.baseTestStateHandler.localstate

	localstate, err := NewLocalstate(
		NewMemStorage(baseLocalstate.Storage().Encoders(), baseLocalstate.Storage().Encoder()),
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

	var suffrage Suffrage
	{
		nodes := make([]Node, len(all))
		for i, o := range all {
			nodes[i] = o.Node()
		}

		suffrage = NewFixedSuffrage(t.localstate.Node(), nodes)
	}

	lastHeight := Height(10)
	bg, err := NewDummyBlocksV0Generator(t.localstate, lastHeight, suffrage, all)
	t.NoError(err)

	t.NoError(bg.Generate(true))

	for i := int64(0); i < lastHeight.Int64(); i++ {
		hashes := map[valuehash.Hash]struct{}{}
		for nodeid, l := range all {
			block, err := l.Storage().BlockByHeight(Height(i))
			t.NoError(err, "node=%d height=%d", nodeid, i)
			t.NotNil(block, "node=%d height=%d", nodeid, i)
			t.NoError(block.IsValid(nil))

			hashes[block.Hash()] = struct{}{}
		}

		t.Equal(1, len(hashes), "check block hashes are matched")
	}
}

func TestBlockV0DummyGenerator(t *testing.T) {
	suite.Run(t, new(testBlockV0DummyGenerator))
}
