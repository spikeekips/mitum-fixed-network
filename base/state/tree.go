package state

import (
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/tree"
)

var (
	FixedTreeNodeType   = hint.Type("state-fixedtree-node")
	FixedTreeNodeHint   = hint.NewHint(FixedTreeNodeType, "v0.0.1")
	FixedTreeNodeHinter = FixedTreeNode{
		BaseFixedTreeNode: tree.BaseFixedTreeNode{BaseHinter: hint.NewBaseHinter(FixedTreeNodeHint)},
	}
)

type FixedTreeNode struct {
	tree.BaseFixedTreeNode
}

func NewFixedTreeNode(index uint64, key []byte) FixedTreeNode {
	return FixedTreeNode{
		BaseFixedTreeNode: tree.NewBaseFixedTreeNode(FixedTreeNodeHint, index, key),
	}
}
