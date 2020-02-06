package mitum

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type testRoundrobinSuffrage struct {
	suite.Suite
}

func (t *testRoundrobinSuffrage) localState() *LocalState {
	localNode := RandomLocalNode("local", nil)
	localState := NewLocalState(localNode, NewLocalPolicy())

	return localState
}

func (t *testRoundrobinSuffrage) TestNew() {
	localState := t.localState()
	sf := NewRoundrobinSuffrage(localState, 10)
	t.NotNil(sf)

	t.Implements((*Suffrage)(nil), sf)
}

func (t *testRoundrobinSuffrage) TestActingSuffrage() {
	localState := t.localState()
	_, _ = localState.Policy().SetNumberOfActingSuffrageNodes(3)

	nodes := []Node{
		RandomLocalNode("n0", nil),
		RandomLocalNode("n1", nil),
		RandomLocalNode("n2", nil),
		RandomLocalNode("n3", nil),
		RandomLocalNode("n4", nil),
	}
	t.NoError(localState.Nodes().Add(nodes...))

	sf := NewRoundrobinSuffrage(localState, 10)

	af := sf.Acting(Height(33), Round(0))
	t.NotNil(af)
	t.Equal(int(localState.Policy().NumberOfActingSuffrageNodes()), len(af.nodes))

	expectedProposer := nodes[2]
	t.True(expectedProposer.Address().Equal(af.Proposer().Address()))

	expected := nodes[2:5]
	for _, n := range af.nodes {
		var found bool
		for _, e := range expected {
			if e.Address().Equal(n.Address()) {
				found = true
				break
			}
		}
		t.True(found)
	}

	t.False(sf.IsActing(Height(33), Round(0), nodes[0].Address()))
	t.False(sf.IsActing(Height(33), Round(0), nodes[1].Address()))
	t.True(sf.IsActing(Height(33), Round(0), nodes[2].Address()))
	t.True(sf.IsProposer(Height(33), Round(0), nodes[2].Address()))
}

func (t *testRoundrobinSuffrage) TestActingSuffrageNotSufficient() {
	localState := t.localState()
	_, _ = localState.Policy().SetNumberOfActingSuffrageNodes(4)

	nodes := []Node{
		RandomLocalNode("n0", nil),
		RandomLocalNode("n1", nil),
	}
	t.NoError(localState.Nodes().Add(nodes...))

	sf := NewRoundrobinSuffrage(localState, 10)

	af := sf.Acting(Height(33), Round(0))
	t.NotNil(af)
	t.Equal(len(nodes)+1, len(af.nodes))

	expectedProposer := localState.Node()
	t.True(expectedProposer.Address().Equal(af.Proposer().Address()))

	expected := []Node{localState.Node()}
	expected = append(expected, nodes...)
	for _, n := range af.nodes {
		var found bool
		for _, e := range expected {
			if e.Address().Equal(n.Address()) {
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
