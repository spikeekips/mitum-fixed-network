package tree

import (
	"github.com/spikeekips/avl"
	avlHashable "github.com/spikeekips/avl/hashable"

	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

var (
	AVLTreeType = hint.MustNewType(0x10, 0x00, "avltree")
	AVLTreeHint = hint.MustHint(AVLTreeType, "0.0.1")
)

type AVLTree struct {
	*avl.Tree
}

func NewAVLTree(tree *avl.Tree) (*AVLTree, error) {
	return &AVLTree{Tree: tree}, nil
}

func (at AVLTree) Hint() hint.Hint {
	return AVLTreeHint
}

func (at *AVLTree) Root() Node {
	return at.Tree.Root().(Node)
}

func (at *AVLTree) RootHash() (valuehash.Hash, error) {
	return valuehash.LoadSHA256FromBytes(at.Root().Hash())
}

func (at *AVLTree) IsValid() error {
	if err := at.Tree.IsValid(); err != nil {
		return err
	}

	var root Node
	if h, ok := at.Tree.Root().(avlHashable.HashableMutableNode); !ok {
		return isvalid.InvalidError.Errorf("root node is not hashable.HashableMutableNode type: %T", at.Tree.Root())
	} else if r, ok := h.(Node); !ok {
		return isvalid.InvalidError.Errorf("root node is not Node type: %T", at.Tree.Root())
	} else {
		root = r
	}

	if len(root.Hash()) < 1 {
		if err := avlHashable.SetTreeNodeHash(root.(avlHashable.HashableMutableNode), GenerateNodeHash); err != nil {
			return nil
		}
	}

	return nil
}

func (at *AVLTree) Empty() bool {
	return at.Tree == nil
}

func (at *AVLTree) Traverse(f func(Node) (keep bool, err error)) error {
	return at.Tree.Traverse(func(node avl.Node) (bool, error) {
		return f(node.(Node))
	})
}

func (at *AVLTree) Get(key []byte) (Node, error) {
	node, err := at.Tree.Get(key)
	if err != nil {
		return nil, err
	}

	return node.(Node), nil
}

func (at *AVLTree) GetWithParents(key []byte) (Node, []Node, error) {
	node, parents, err := at.Tree.GetWithParents(key)
	if err != nil {
		return nil, nil, err
	}

	nn := make([]Node, len(parents))
	for i := range parents {
		nn[i] = parents[i].(Node)
	}

	return node.(Node), nn, nil
}

func GenerateNodeHash(node avlHashable.HashableNode) ([]byte, error) {
	e := util.ConcatBytesSlice(
		node.Key(),
		util.Int64ToBytes(int64(node.Height())),
		node.ValueHash(),
		node.LeftHash(),
		node.RightHash(),
	)

	return valuehash.NewSHA256(e).Bytes(), nil
}
