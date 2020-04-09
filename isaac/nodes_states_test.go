package isaac

import (
	"sort"
	"strings"
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/stretchr/testify/suite"
)

type testNodesState struct {
	suite.Suite
	localNode *LocalNode
}

func (t *testNodesState) SetupSuite() {
	t.localNode = RandomLocalNode("local", nil)
}

func (t *testNodesState) TestEmpty() {
	var nodes []Node
	ns := NewNodesState(t.localNode, nodes)
	t.Equal(0, ns.Len())
}

func (t *testNodesState) TestDuplicatedAddress() {
	nodes := []Node{
		RandomLocalNode("n0", nil),
		RandomLocalNode("n0", nil), // will be ignored
		RandomLocalNode("n1", nil),
	}

	ns := NewNodesState(t.localNode, nodes)
	t.Equal(len(nodes)-1, ns.Len())
}

func (t *testNodesState) TestAdd() {
	nodes := []Node{
		RandomLocalNode("n0", nil),
		RandomLocalNode("n1", nil),
	}

	ns := NewNodesState(t.localNode, nodes)

	{ // add, but same Address
		err := ns.Add(RandomLocalNode("n1", nil))
		t.Contains(err.Error(), "already exists")
		t.Equal(len(nodes), ns.Len())
	}

	newNode := RandomLocalNode("n2", nil)
	err := ns.Add(newNode)
	t.NoError(err)
	t.Equal(len(nodes)+1, ns.Len())

	var added []Node
	ns.Traverse(func(n Node) bool {
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

func (t *testNodesState) TestRemove() {
	nodes := []Node{
		RandomLocalNode("n0", nil),
		RandomLocalNode("n1", nil),
		RandomLocalNode("n2", nil),
	}

	ns := NewNodesState(t.localNode, nodes)

	{ // try to remove, but nothing
		err := ns.Remove(base.NewShortAddress("hehe"))
		t.Contains(err.Error(), "does not exist")
		t.Equal(len(nodes), ns.Len())
	}

	err := ns.Remove(nodes[2].Address())
	t.NoError(err)
	t.Equal(len(nodes)-1, ns.Len())

	var removed []Node
	ns.Traverse(func(n Node) bool {
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

func (t *testNodesState) TestTraverse() {
	nodes := []Node{
		RandomLocalNode("n0", nil),
		RandomLocalNode("n1", nil),
		RandomLocalNode("n2", nil),
	}

	ns := NewNodesState(t.localNode, nodes)

	{ // all
		var traversed []Node
		ns.Traverse(func(n Node) bool {
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
		var traversed []Node
		ns.Traverse(func(n Node) bool {
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

func TestNodesState(t *testing.T) {
	suite.Run(t, new(testNodesState))
}
