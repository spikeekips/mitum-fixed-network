package network

import (
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/stretchr/testify/suite"
)

type testNodepool struct {
	suite.Suite
	local *LocalNode
}

func (t *testNodepool) SetupSuite() {
	t.local = RandomLocalNode("local", nil)
}

func (t *testNodepool) TestEmpty() {
	ns := NewNodepool(t.local)
	t.Equal(1, ns.Len())
}

func (t *testNodepool) TestDuplicatedAddress() {
	nodes := []Node{
		RandomLocalNode("n0", nil),
		RandomLocalNode("n0", nil), // will be ignored
		RandomLocalNode("n1", nil),
	}

	ns := NewNodepool(t.local)
	err := ns.Add(nodes...)
	t.Contains(err.Error(), "duplicated Address found")
}

func (t *testNodepool) TestAdd() {
	nodes := []Node{
		RandomLocalNode("n0", nil),
		RandomLocalNode("n1", nil),
	}

	ns := NewNodepool(t.local)
	t.NoError(ns.Add(nodes...))

	{ // add, but same Address
		err := ns.Add(RandomLocalNode("n1", nil))
		t.Contains(err.Error(), "already exists")
		t.Equal(len(nodes)+1, ns.Len())
	}

	newNode := RandomLocalNode("n2", nil)
	err := ns.Add(newNode)
	t.NoError(err)
	t.Equal(len(nodes)+2, ns.Len())

	for _, n := range nodes {
		t.True(ns.Exists(n.Address()))
	}
	t.True(ns.Exists(newNode.Address()))
}

func (t *testNodepool) TestAddSameWithLocal() {
	ns := NewNodepool(t.local)

	err := ns.Add(t.local)
	t.Contains(err.Error(), "same Address already exists")
}

func (t *testNodepool) TestAddDuplicated() {
	ns := NewNodepool(t.local)

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

	ns := NewNodepool(t.local)
	t.NoError(ns.Add(nodes...))

	{ // try to remove, but nothing
		err := ns.Remove(base.RandomStringAddress())
		t.Contains(err.Error(), "does not exist")
		t.Equal(len(nodes)+1, ns.Len())
	}

	err := ns.Remove(nodes[2].Address())
	t.NoError(err)
	t.Equal(len(nodes), ns.Len())

	t.False(ns.Exists(nodes[2].Address()))
}

func (t *testNodepool) TestTraverse() {
	nodes := []Node{
		RandomLocalNode("n0", nil),
		RandomLocalNode("n1", nil),
		RandomLocalNode("n2", nil),
	}

	ns := NewNodepool(t.local)
	t.NoError(ns.Add(nodes...))

	var traversed []Node
	ns.Traverse(func(n Node) bool {
		traversed = append(traversed, n)
		return true
	})

	t.Equal(len(nodes)+1, len(traversed))
	for _, n := range traversed {
		t.True(ns.Exists(n.Address()))
	}
}

func TestNodepool(t *testing.T) {
	suite.Run(t, new(testNodepool))
}
