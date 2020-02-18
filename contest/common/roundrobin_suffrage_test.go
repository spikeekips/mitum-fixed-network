package common

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/isaac"
)

type testRoundrobinSuffrage struct {
	suite.Suite
}

func (t *testRoundrobinSuffrage) localState() *isaac.LocalState {
	localNode := isaac.RandomLocalNode("local", nil)
	localState, err := isaac.NewLocalState(nil, localNode)
	t.NoError(err)

	return localState
}

func (t *testRoundrobinSuffrage) TestNew() {
	localState := t.localState()
	sf := NewRoundrobinSuffrage(localState, 10)
	t.NotNil(sf)

	t.Implements((*isaac.Suffrage)(nil), sf)
}

func (t *testRoundrobinSuffrage) TestActingSuffrage() {
	localState := t.localState()
	_, _ = localState.Policy().SetNumberOfActingSuffrageNodes(3)

	nodes := []isaac.Node{
		isaac.RandomLocalNode("n0", nil),
		isaac.RandomLocalNode("n1", nil),
		isaac.RandomLocalNode("n2", nil),
		isaac.RandomLocalNode("n3", nil),
		isaac.RandomLocalNode("n4", nil),
	}
	t.NoError(localState.Nodes().Add(nodes...))

	sf := NewRoundrobinSuffrage(localState, 10)

	af := sf.Acting(isaac.Height(33), isaac.Round(0))
	t.NotNil(af)
	t.Equal(int(localState.Policy().NumberOfActingSuffrageNodes()), len(af.Nodes()))

	expectedProposer := nodes[2]
	t.True(expectedProposer.Address().Equal(af.Proposer().Address()))

	expected := nodes[2:5]
	for _, n := range af.Nodes() {
		var found bool
		for _, e := range expected {
			if e.Address().Equal(n.Address()) {
				found = true
				break
			}
		}
		t.True(found)
	}

	t.False(sf.IsActing(isaac.Height(33), isaac.Round(0), nodes[0].Address()))
	t.False(sf.IsActing(isaac.Height(33), isaac.Round(0), nodes[1].Address()))
	t.True(sf.IsActing(isaac.Height(33), isaac.Round(0), nodes[2].Address()))
	t.True(sf.IsProposer(isaac.Height(33), isaac.Round(0), nodes[2].Address()))
}

func (t *testRoundrobinSuffrage) TestActingSuffrageNotSufficient() {
	localState := t.localState()
	_, _ = localState.Policy().SetNumberOfActingSuffrageNodes(4)

	nodes := []isaac.Node{
		isaac.RandomLocalNode("n0", nil),
		isaac.RandomLocalNode("n1", nil),
	}
	t.NoError(localState.Nodes().Add(nodes...))

	sf := NewRoundrobinSuffrage(localState, 10)

	af := sf.Acting(isaac.Height(33), isaac.Round(0))
	t.NotNil(af)
	t.Equal(len(nodes)+1, len(af.Nodes()))

	expectedProposer := localState.Node()
	t.True(expectedProposer.Address().Equal(af.Proposer().Address()))

	expected := []isaac.Node{localState.Node()}
	expected = append(expected, nodes...)
	for _, n := range af.Nodes() {
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
