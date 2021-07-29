package network

import (
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/util"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testNodepool struct {
	suite.Suite
	local *node.Local
}

func (t *testNodepool) SetupSuite() {
	t.local = node.RandomLocal("local")
}

func (t *testNodepool) TestEmpty() {
	ns := NewNodepool(t.local, nil)
	t.Equal(1, ns.Len())
}

func (t *testNodepool) TestDuplicatedAddress() {
	ns := NewNodepool(t.local, nil)

	t.NoError(ns.Add(node.RandomLocal("n0"), nil))
	t.NoError(ns.Add(node.RandomLocal("n1"), nil))

	err := ns.Add(node.RandomLocal("n0"), nil) // will be ignored
	t.True(xerrors.Is(err, util.FoundError))
}

func (t *testNodepool) TestAdd() {
	nodes := []base.Node{
		node.RandomLocal("n0"),
		node.RandomLocal("n1"),
	}

	ns := NewNodepool(t.local, nil)
	for i := range nodes {
		t.NoError(ns.Add(nodes[i], nil))
	}

	{ // add, but same Address
		err := ns.Add(node.RandomLocal("n1"), nil)
		t.Contains(err.Error(), "already exists")
		t.Equal(len(nodes)+1, ns.Len())
	}

	newNode := node.RandomLocal("n2")
	err := ns.Add(newNode, nil)
	t.NoError(err)
	t.Equal(len(nodes)+2, ns.Len())

	for _, n := range nodes {
		t.True(ns.Exists(n.Address()))
	}
	t.True(ns.Exists(newNode.Address()))
}

func (t *testNodepool) TestAddSameWithLocal() {
	ns := NewNodepool(t.local, nil)

	err := ns.Add(t.local, nil)
	t.True(xerrors.Is(err, util.FoundError))
}

func (t *testNodepool) TestAddDuplicated() {
	ns := NewNodepool(t.local, nil)

	newNode := node.RandomLocal("n2")
	t.NoError(ns.Add(newNode, nil))

	err := ns.Add(newNode, nil)
	t.True(xerrors.Is(err, util.FoundError))
}

func (t *testNodepool) TestRemove() {
	nodes := []base.Node{
		node.RandomLocal("n0"),
		node.RandomLocal("n1"),
		node.RandomLocal("n2"),
	}

	ns := NewNodepool(t.local, nil)
	for i := range nodes {
		t.NoError(ns.Add(nodes[i], nil))
	}

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
	nodes := []base.Node{
		node.RandomLocal("n0"),
		node.RandomLocal("n1"),
		node.RandomLocal("n2"),
	}

	ns := NewNodepool(t.local, nil)
	for i := range nodes {
		t.NoError(ns.Add(nodes[i], nil))
	}

	var tnodes []base.Node
	var tchs []Channel
	ns.Traverse(func(n base.Node, ch Channel) bool {
		tnodes = append(tnodes, n)
		tchs = append(tchs, ch)
		return true
	})

	t.Equal(len(nodes)+1, len(tnodes))
	t.Equal(len(nodes)+1, len(tchs))
	for i := range tnodes {
		bn := tnodes[i]

		t.True(ns.Exists(bn.Address()))

		an, ch, found := ns.Node(bn.Address())
		t.True(found)
		t.Nil(ch)

		t.True(an.Address().Equal(bn.Address()))
		t.True(an.Publickey().Equal(bn.Publickey()))
	}
}

func TestNodepool(t *testing.T) {
	suite.Run(t, new(testNodepool))
}
