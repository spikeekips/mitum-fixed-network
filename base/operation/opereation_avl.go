package operation

import (
	"github.com/spikeekips/avl"
	avlHashable "github.com/spikeekips/avl/hashable"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/tree"
	"github.com/spikeekips/mitum/util/hint"
)

var (
	OperationAVLNodeType        = hint.MustNewType(0x01, 0x52, "operation-avlnode")
	OperationAVLNodeHint        = hint.MustHint(OperationAVLNodeType, "0.0.1")
	OperationAVLNodeMutableType = hint.MustNewType(0x01, 0x53, "operation-avlnode-mutable")
	OperationAVLNodeMutableHint = hint.MustHint(OperationAVLNodeMutableType, "0.0.1")
)

type OperationAVLNode struct {
	key       []byte
	height    int16
	left      []byte
	leftHash  []byte
	right     []byte
	rightHash []byte
	h         []byte
	op        Operation
}

func NewOperationAVLNode(n *OperationAVLNodeMutable) OperationAVLNode {
	return OperationAVLNode{
		key:       n.Key(),
		height:    n.Height(),
		left:      n.LeftKey(),
		leftHash:  n.LeftHash(),
		right:     n.RightKey(),
		rightHash: n.RightHash(),
		h:         n.Hash(),
		op:        n.Operation(),
	}
}

func (em OperationAVLNode) Hint() hint.Hint {
	return OperationAVLNodeHint
}

func (em OperationAVLNode) Key() []byte {
	return em.key
}

func (em OperationAVLNode) Height() int16 {
	return em.height
}

func (em OperationAVLNode) LeftKey() []byte {
	return em.left
}

func (em OperationAVLNode) RightKey() []byte {
	return em.right
}

func (em OperationAVLNode) Hash() []byte {
	return em.h
}

func (em OperationAVLNode) LeftHash() []byte {
	return em.leftHash
}

func (em OperationAVLNode) RightHash() []byte {
	return em.rightHash
}

func (em OperationAVLNode) ValueHash() []byte {
	return em.op.Hash().Bytes()
}

func (em OperationAVLNode) Operation() Operation {
	return em.op
}

func (em OperationAVLNode) Immutable() tree.Node {
	return em
}

type OperationAVLNodeMutable struct {
	key    []byte // Operation.Hash().Bytes()
	height int16
	left   avlHashable.HashableMutableNode
	right  avlHashable.HashableMutableNode
	h      []byte
	op     Operation // Operation
}

func NewOperationAVLNodeMutable(op Operation) *OperationAVLNodeMutable {
	return &OperationAVLNodeMutable{key: []byte(op.Hash().String()), op: op}
}

func (em OperationAVLNodeMutable) Hint() hint.Hint {
	return OperationAVLNodeMutableHint
}

func (em *OperationAVLNodeMutable) Key() []byte {
	return em.key
}

func (em *OperationAVLNodeMutable) Height() int16 {
	return em.height
}

func (em *OperationAVLNodeMutable) SetHeight(height int16) error {
	if height < 0 {
		return xerrors.Errorf("height must be greater than zero; height=%d", height)
	}

	em.height = height

	return nil
}

func (em *OperationAVLNodeMutable) Left() avl.MutableNode {
	return em.left
}

func (em *OperationAVLNodeMutable) LeftKey() []byte {
	if em.left == nil {
		return nil
	}

	return em.left.Key()
}

func (em *OperationAVLNodeMutable) SetLeft(node avl.MutableNode) error {
	if node == nil {
		em.left = nil
		return nil
	}

	if avl.EqualKey(em.key, node.Key()) {
		return xerrors.Errorf("left is same node; key=%v", em.key)
	}

	m, ok := node.(avlHashable.HashableMutableNode)
	if !ok {
		return xerrors.Errorf("not HashableMutableNode; %T", node)
	}

	em.left = m

	return nil
}

func (em *OperationAVLNodeMutable) Right() avl.MutableNode {
	return em.right
}

func (em *OperationAVLNodeMutable) RightKey() []byte {
	if em.right == nil {
		return nil
	}

	return em.right.Key()
}

func (em *OperationAVLNodeMutable) SetRight(node avl.MutableNode) error {
	if node == nil {
		em.right = nil
		return nil
	}

	if avl.EqualKey(em.key, node.Key()) {
		return xerrors.Errorf("right is same node; key=%v", em.key)
	}

	m, ok := node.(avlHashable.HashableMutableNode)
	if !ok {
		return xerrors.Errorf("not HashableMutableNode; %T", node)
	}

	em.right = m

	return nil
}

func (em *OperationAVLNodeMutable) Merge(avl.MutableNode) error {
	return nil
}

func (em *OperationAVLNodeMutable) Hash() []byte {
	return em.h
}

func (em *OperationAVLNodeMutable) SetHash(h []byte) error {
	em.h = h

	return nil
}

func (em *OperationAVLNodeMutable) ResetHash() {
	em.h = nil
}

func (em *OperationAVLNodeMutable) LeftHash() []byte {
	if em.left == nil {
		return nil
	}

	return em.left.Hash()
}

func (em *OperationAVLNodeMutable) RightHash() []byte {
	if em.right == nil {
		return nil
	}

	return em.right.Hash()
}

func (em *OperationAVLNodeMutable) ValueHash() []byte {
	return em.op.Hash().Bytes()
}

func (em *OperationAVLNodeMutable) Operation() Operation {
	return em.op
}

func (em *OperationAVLNodeMutable) Immutable() tree.Node {
	return NewOperationAVLNode(em)
}
