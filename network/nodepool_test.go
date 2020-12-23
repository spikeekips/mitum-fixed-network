package network

import (
	"sort"
	"strings"
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/stretchr/testify/suite"
)

type testNodepool struct {
	suite.Suite
	localNode *LocalNode
}

func (t *testNodepool) SetupSuite() {
	t.localNode = RandomLocalNode("local", nil)
}

func (t *testNodepool) TestEmpty() {
	ns := NewNodepool(t.localNode)
	t.Equal(0, ns.Len())
}

func (t *testNodepool) TestDuplicatedAddress() {
	nodes := []Node{
		RandomLocalNode("n0", nil),
		RandomLocalNode("n0", nil), // will be ignored
		RandomLocalNode("n1", nil),
	}

	ns := NewNodepool(t.localNode)
	err := ns.Add(nodes...)
	t.Contains(err.Error(), "duplicated Address found")
}

func (t *testNodepool) TestAdd() {
	nodes := []Node{
		RandomLocalNode("n0", nil),
		RandomLocalNode("n1", nil),
	}

	ns := NewNodepool(t.localNode)
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

func (t *testNodepool) TestAddSameWithLocal() {
	ns := NewNodepool(t.localNode)

	err := ns.Add(t.localNode)
	t.Contains(err.Error(), "local node can not be added")
}

func (t *testNodepool) TestAddDuplicated() {
	ns := NewNodepool(t.localNode)

	newNode := RandomLocalNode("n2", nil)
	err := ns.Add(newNode, newNode)
	t.Contains(err.Error(), "duplicated Address found")
}

func (t *testNodepool) TestRemove() {
	nodes := []Node{
		RandomLocalNode("n0", nil),
		RandomLocalNode("n1", nil),
		RandomLocalNode("n2", nil),
	}

	ns := NewNodepool(t.localNode)
	t.NoError(ns.Add(nodes...))

	{ // try to remove, but nothing
		err := ns.Remove(base.RandomStringAddress())
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

func (t *testNodepool) TestTraverse() {
	nodes := []Node{
		RandomLocalNode("n0", nil),
		RandomLocalNode("n1", nil),
		RandomLocalNode("n2", nil),
	}

	ns := NewNodepool(t.localNode)
	t.NoError(ns.Add(nodes...))

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

func TestNodepool(t *testing.T) {
	suite.Run(t, new(testNodepool))
}
