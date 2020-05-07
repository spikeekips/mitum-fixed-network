package contestlib

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/isaac"
)

type testRoundrobinSuffrage struct {
	suite.Suite
}

func (t *testRoundrobinSuffrage) localstate() *isaac.Localstate {
	localNode := isaac.RandomLocalNode("local", nil)
	localstate, err := isaac.NewLocalstate(nil, localNode, isaac.TestNetworkID)
	t.NoError(err)

	return localstate
}

func (t *testRoundrobinSuffrage) TestNew() {
	localstate := t.localstate()
	sf := NewRoundrobinSuffrage(localstate, 10)
	t.NotNil(sf)

	t.Implements((*base.Suffrage)(nil), sf)
}

func (t *testRoundrobinSuffrage) TestActingSuffrage() {
	localstate := t.localstate()
	_, _ = localstate.Policy().SetNumberOfActingSuffrageNodes(3)

	nodes := []isaac.Node{
		isaac.RandomLocalNode("n0", nil),
		isaac.RandomLocalNode("n1", nil),
		isaac.RandomLocalNode("n2", nil),
		isaac.RandomLocalNode("n3", nil),
		isaac.RandomLocalNode("n4", nil),
	}
	t.NoError(localstate.Nodes().Add(nodes...))

	sf := NewRoundrobinSuffrage(localstate, 10)

	af := sf.Acting(base.Height(33), base.Round(0))
	t.NotNil(af)
	t.Equal(int(localstate.Policy().NumberOfActingSuffrageNodes()), len(af.Nodes()))

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
	localstate := t.localstate()
	_, _ = localstate.Policy().SetNumberOfActingSuffrageNodes(4)

	nodes := []isaac.Node{
		isaac.RandomLocalNode("n0", nil),
		isaac.RandomLocalNode("n1", nil),
	}
	t.NoError(localstate.Nodes().Add(nodes...))

	sf := NewRoundrobinSuffrage(localstate, 10)

	af := sf.Acting(base.Height(33), base.Round(0))
	t.NotNil(af)
	t.Equal(len(nodes)+1, len(af.Nodes()))

	expectedProposer := localstate.Node()
	t.True(expectedProposer.Address().Equal(af.Proposer()))

	expected := []isaac.Node{localstate.Node()}
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
