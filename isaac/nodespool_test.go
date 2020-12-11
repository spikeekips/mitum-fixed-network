package isaac

import (
	"sort"
	"strings"
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/network"
	"github.com/stretchr/testify/suite"
)

type testNodesPool struct {
	suite.Suite
	localNode *LocalNode
}

func (t *testNodesPool) SetupSuite() {
	t.localNode = RandomLocalNode("local", nil)
}

func (t *testNodesPool) TestEmpty() {
	ns := NewNodesPool(t.localNode)
	t.Equal(0, ns.Len())
}

func (t *testNodesPool) TestDuplicatedAddress() {
	nodes := []network.Node{
		RandomLocalNode("n0", nil),
		RandomLocalNode("n0", nil), // will be ignored
		RandomLocalNode("n1", nil),
	}

	ns := NewNodesPool(t.localNode)
	err := ns.Add(nodes...)
	t.Contains(err.Error(), "duplicated Address found")
}

func (t *testNodesPool) TestAdd() {
	nodes := []network.Node{
		RandomLocalNode("n0", nil),
		RandomLocalNode("n1", nil),
	}

	ns := NewNodesPool(t.localNode)
	t.NoError(ns.Add(nodes...))

	{ // add, but same Address
		err := ns.Add(RandomLocalNode("n1", nil))
		t.Contains(err.Error(), "already exists")
		t.Equal(len(nodes), ns.Len())
	}

	newNode := RandomLocalNode("n2", nil)
	err := ns.Add(newNode)
	t.NoError(err)
	t.Equal(len(nodes)+1, ns.Len())

	var added []network.Node
	ns.Traverse(func(n network.Node) bool {
		added = append(added, n)
		return true
	})
	sort.Slice(
		added,
		func(i, j int) bool {
			return strings.Compare(
				added[i].Address().String(),
				added[j].Address().String(),
			) < 0
		},
	)

	for i, n := range nodes {
		t.True(n.Address().Equal(added[i].Address()))
	}
	t.True(newNode.Address().Equal(added[2].Address()))
}

func (t *testNodesPool) TestAddSameWithLocal() {
	ns := NewNodesPool(t.localNode)

	err := ns.Add(t.localNode)
	t.Contains(err.Error(), "local node can not be added")
}

func (t *testNodesPool) TestAddDuplicated() {
	ns := NewNodesPool(t.localNode)

	newNode := RandomLocalNode("n2", nil)
	err := ns.Add(newNode, newNode)
	t.Contains(err.Error(), "duplicated Address found")
}

func (t *testNodesPool) TestRemove() {
	nodes := []network.Node{
		RandomLocalNode("n0", nil),
		RandomLocalNode("n1", nil),
		RandomLocalNode("n2", nil),
	}

	ns := NewNodesPool(t.localNode)
	t.NoError(ns.Add(nodes...))

	{ // try to remove, but nothing
		err := ns.Remove(base.RandomStringAddress())
		t.Contains(err.Error(), "does not exist")
		t.Equal(len(nodes), ns.Len())
	}

	err := ns.Remove(nodes[2].Address())
	t.NoError(err)
	t.Equal(len(nodes)-1, ns.Len())

	var removed []network.Node
	ns.Traverse(func(n network.Node) bool {
		removed = append(removed, n)
		return true
	})
	sort.Slice(
		removed,
		func(i, j int) bool {
			return strings.Compare(
				removed[i].Address().String(),
				removed[j].Address().String(),
			) < 0
		},
	)

	for i, n := range removed {
		t.True(nodes[i].Address().Equal(n.Address()))
	}
}

func (t *testNodesPool) TestTraverse() {
	nodes := []network.Node{
		RandomLocalNode("n0", nil),
		RandomLocalNode("n1", nil),
		RandomLocalNode("n2", nil),
	}

	ns := NewNodesPool(t.localNode)
	t.NoError(ns.Add(nodes...))

	{ // all
		var traversed []network.Node
		ns.Traverse(func(n network.Node) bool {
			traversed = append(traversed, n)
			return true
		})
		sort.Slice(
			traversed,
			func(i, j int) bool {
				return strings.Compare(
					traversed[i].Address().String(),
					traversed[j].Address().String(),
				) < 0
			},
		)

		for i, n := range traversed {
			t.True(nodes[i].Address().Equal(n.Address()))
		}
	}

	{ // only first one
		var traversed []network.Node
		ns.Traverse(func(n network.Node) bool {
			if n.Address().Equal(nodes[1].Address()) {
				traversed = append(traversed, n)
				return false
			}
			return true
		})

		t.Equal(1, len(traversed))
		t.True(traversed[0].Address().Equal(nodes[1].Address()))
	}
}

func TestNodesPool(t *testing.T) {
	suite.Run(t, new(testNodesPool))
}
