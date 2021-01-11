package process

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testRoundrobinSuffrage struct {
	suite.Suite
}

func (t *testRoundrobinSuffrage) local() *isaac.Local {
	localNode := network.RandomLocalNode("local", nil)
	local, err := isaac.NewLocal(nil, nil, localNode, isaac.TestNetworkID)
	t.NoError(err)

	t.NoError(local.Initialize())

	return local
}

func (t *testRoundrobinSuffrage) TestNew() {
	local := t.local()
	sf := NewRoundrobinSuffrage(local, 10, 1, nil)
	t.NotNil(sf)

	t.Implements((*base.Suffrage)(nil), sf)
}

func (t *testRoundrobinSuffrage) TestActingSuffrage() {
	local := t.local()

	var na uint = 3

	nodes := []network.Node{
		network.RandomLocalNode("n0", nil),
		network.RandomLocalNode("n1", nil),
		network.RandomLocalNode("n2", nil),
		network.RandomLocalNode("n3", nil),
		network.RandomLocalNode("n4", nil),
	}
	t.NoError(local.Nodes().Add(nodes...))

	sf := NewRoundrobinSuffrage(local, 10, na, func(base.Height) (valuehash.Hash, error) {
		return valuehash.NewBytes([]byte("showme 5")), nil
	})

	af, err := sf.Acting(base.Height(33), base.Round(0))
	t.NoError(err)
	t.NotNil(af)
	t.Equal(int(na), len(af.Nodes()))

	expectedProposer := nodes[2]
	t.True(expectedProposer.Address().Equal(af.Proposer()))

	expected := nodes[2:5]
	for _, n := range af.Nodes() {
		var found bool
		for _, e := range expected {
			if e.Address().Equal(n) {
				found = true
				break
			}
		}
		t.True(found)
	}

	t.False(sf.IsActing(base.Height(33), base.Round(0), nodes[0].Address()))
	t.False(sf.IsActing(base.Height(33), base.Round(0), nodes[1].Address()))
	t.True(sf.IsActing(base.Height(33), base.Round(0), nodes[2].Address()))
	t.True(sf.IsProposer(base.Height(33), base.Round(0), nodes[2].Address()))
}

func (t *testRoundrobinSuffrage) TestActingSuffrageNotSufficient() {
	local := t.local()

	var na uint = 4

	nodes := []network.Node{
		network.RandomLocalNode("n0", nil),
		network.RandomLocalNode("n1", nil),
	}
	t.NoError(local.Nodes().Add(nodes...))

	sf := NewRoundrobinSuffrage(local, 10, na, nil)

	af, err := sf.Acting(base.Height(33), base.Round(0))
	t.NoError(err)
	t.NotNil(af)
	t.Equal(len(nodes)+1, len(af.Nodes()))

	expectedProposer := local.Node()
	t.True(expectedProposer.Address().Equal(af.Proposer()))

	expected := []network.Node{local.Node()}
	expected = append(expected, nodes...)
	for _, n := range af.Nodes() {
		var found bool
		for _, e := range expected {
			if e.Address().Equal(n) {
				found = true
				break
			}
		}
		t.True(found)
	}
}

func TestRoundrobinSuffrage(t *testing.T) {
	suite.Run(t, new(testRoundrobinSuffrage))
}
