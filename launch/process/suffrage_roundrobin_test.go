package process

import (
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testRoundrobinSuffrage struct {
	suite.Suite
}

func (t *testRoundrobinSuffrage) nodes(n int) []base.Address {
	nodes := make([]base.Address, n)
	for i := 0; i < n; i++ {
		nodes[i] = base.RandomStringAddress()
	}

	return nodes
}

func (t *testRoundrobinSuffrage) TestNew() {
	nodes := t.nodes(3)
	sf, err := NewRoundrobinSuffrage(nodes, 3, 1, nil)
	t.NoError(err)
	t.NotNil(sf)

	t.Implements((*base.Suffrage)(nil), sf)
}

func (t *testRoundrobinSuffrage) TestActingSuffrage() {
	var na uint = 3

	nodes := t.nodes(5)

	sf, err := NewRoundrobinSuffrage(nodes, na, 10, func(base.Height) (valuehash.Hash, error) {
		return valuehash.NewBytes([]byte("showme 5")), nil
	})
	t.NoError(err)

	af, err := sf.Acting(base.Height(33), base.Round(0))
	t.NoError(err)
	t.NotNil(af)
	t.Equal(int(na), len(af.Nodes()))

	expectedProposer := nodes[2]
	t.True(expectedProposer.Equal(af.Proposer()))

	expected := nodes[2:5]
	for _, n := range af.Nodes() {
		var found bool
		for _, e := range expected {
			if e.Equal(n) {
				found = true
				break
			}
		}
		t.True(found)
	}

	t.False(sf.IsActing(base.Height(33), base.Round(0), nodes[0]))
	t.False(sf.IsActing(base.Height(33), base.Round(0), nodes[1]))
	t.True(sf.IsActing(base.Height(33), base.Round(0), nodes[2]))
	t.True(sf.IsProposer(base.Height(33), base.Round(0), nodes[2]))
}

func (t *testRoundrobinSuffrage) TestActingSuffrageNotSufficient() {
	var na uint = 4

	nodes := t.nodes(2)

	_, err := NewRoundrobinSuffrage(nodes, na, 10, nil)
	t.Contains(err.Error(), "under number of acting")
}

func TestRoundrobinSuffrage(t *testing.T) {
	suite.Run(t, new(testRoundrobinSuffrage))
}
