package network

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util"
	"github.com/stretchr/testify/suite"
)

func (np *Nodepool) lenPassthroughs() int {
	var c int
	_ = np.pts.Traverse(func(interface{}, interface{}) bool {
		c++

		return true
	})

	return c
}

type testNodepool struct {
	suite.Suite
	local node.Local
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
	t.True(errors.Is(err, util.FoundError))
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
	t.True(errors.Is(err, util.FoundError))
}

func (t *testNodepool) TestAddDuplicated() {
	ns := NewNodepool(t.local, nil)

	newNode := node.RandomLocal("n2")
	t.NoError(ns.Add(newNode, nil))

	err := ns.Add(newNode, nil)
	t.True(errors.Is(err, util.FoundError))
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

func (t *testNodepool) TestAddPassthrough() {
	ns := NewNodepool(t.local, nil)

	for i := 0; i < 10; i++ {
		ch := NilConnInfoChannel(fmt.Sprintf("ch%d", i))
		t.False(ns.ExistsPassthrough(ch.ConnInfo()))
		t.NoError(ns.SetPassthrough(ch, nil, 0))
		t.True(ns.ExistsPassthrough(ch.ConnInfo()))
	}
}

func (t *testNodepool) TestAddPassthroughButInNodes() {
	ns := NewNodepool(t.local, nil)
	n0 := node.RandomLocal("n0")
	ch0 := NilConnInfoChannel("n0")

	t.NoError(ns.Add(n0, ch0))

	err := ns.SetPassthrough(ch0, nil, 0)
	t.True(errors.Is(err, util.FoundError))
}

func (t *testNodepool) TestAddPassthroughButNoConnInfo() {
	ns := NewNodepool(t.local, nil)
	ch0 := NewDummyChannel(nil)

	err := ns.SetPassthrough(ch0, nil, 0)
	t.Contains(err.Error(), "nil ConnInfo")
}

func (t *testNodepool) TestRemovePassthrough() {
	ns := NewNodepool(t.local, nil)

	chs := make([]Channel, 10)
	for i := 0; i < 10; i++ {
		ch := NilConnInfoChannel(fmt.Sprintf("ch%d", i))
		t.NoError(ns.SetPassthrough(ch, nil, 0))

		chs[i] = ch
	}

	t.NoError(ns.RemovePassthrough(chs[3].ConnInfo().String()))
	err := ns.RemovePassthrough(chs[3].ConnInfo().String())
	t.True(errors.Is(err, util.NotFoundError))

	t.Equal(9, ns.lenPassthroughs())
}

func (t *testNodepool) TestPassthroughs() {
	ns := NewNodepool(t.local, nil)

	ch0 := NilConnInfoChannel("n0")
	t.NoError(ns.SetPassthrough(ch0, nil, 0))

	ch1 := NilConnInfoChannel("n1")
	t.NoError(ns.SetPassthrough(ch1, nil, 0))

	pub := key.NewBasePrivatekey().Publickey()
	sl := NewPassthroughedSeal(seal.NewDummySeal(pub), "")

	passedch := make(chan [2]interface{}, 2)
	ns.Passthroughs(context.Background(), sl, func(sl seal.Seal, ch Channel) {
		passedch <- [2]interface{}{ch.ConnInfo().String(), sl}
	})

	close(passedch)

	passed := map[string]seal.Seal{}
	for i := range passedch {
		pci := i[0].(string)
		psl := i[1].(seal.Seal)

		passed[pci] = psl
	}

	t.Equal(2, len(passed))
	t.True(passed[ch0.ConnInfo().String()].Hash().Equal(sl.Hash()))
	t.True(passed[ch1.ConnInfo().String()].Hash().Equal(sl.Hash()))
}

func (t *testNodepool) TestPassthroughsFilter() {
	ns := NewNodepool(t.local, nil)

	ch0 := NilConnInfoChannel("n0")
	t.NoError(ns.SetPassthrough(ch0, nil, 0))

	ch1 := NilConnInfoChannel("n1")
	t.NoError(ns.SetPassthrough(ch1, func(sl PassthroughedSeal) bool {
		return false
	}, 0))

	pub := key.NewBasePrivatekey().Publickey()
	sl := NewPassthroughedSeal(seal.NewDummySeal(pub), "")

	passedch := make(chan [2]interface{}, 2)
	ns.Passthroughs(context.Background(), sl, func(sl seal.Seal, ch Channel) {
		passedch <- [2]interface{}{ch.ConnInfo().String(), sl}
	})

	close(passedch)

	passed := map[string]seal.Seal{}
	for i := range passedch {
		pci := i[0].(string)
		psl := i[1].(seal.Seal)

		passed[pci] = psl
	}

	t.Equal(1, len(passed))
	t.True(passed[ch0.ConnInfo().String()].Hash().Equal(sl.Hash()))
	_, found := passed[ch1.ConnInfo().String()]
	t.False(found)
}

func (t *testNodepool) TestPassthroughExpire() {
	ns := NewNodepool(t.local, nil)

	ch := NilConnInfoChannel("ne")
	t.NoError(ns.SetPassthrough(ch, nil, 0)) // never expire

	p, found := ns.passthrough(ch.ConnInfo())
	t.True(found)
	t.NotNil(p)

	t.NoError(ns.SetPassthrough(ch, nil, time.Millisecond*10)) // will expire 10 ms
	<-time.After(time.Millisecond * 20)

	p, found = ns.passthrough(ch.ConnInfo())
	t.False(found)
	t.Empty(p)
}

func TestNodepool(t *testing.T) {
	suite.Run(t, new(testNodepool))
}
