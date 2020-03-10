package operation

import (
	"github.com/spikeekips/avl"
	avlHashable "github.com/spikeekips/avl/hashable"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/valuehash"
)

var OperationAVLNodeHint hint.Hint = hint.MustHint(hint.Type{0x09, 0x01}, "0.0.1")

type OperationAVLNode struct {
	key    []byte // Operation.Hash().Bytes()
	height int16
	left   avlHashable.HashableMutableNode
	right  avlHashable.HashableMutableNode
	h      []byte
	op     valuehash.Hash // Operation.Hash()
}

func NewOperationAVLNode(op Operation) *OperationAVLNode {
	return &OperationAVLNode{key: []byte(op.Hash().String()), op: op.Hash()}
}

func (em OperationAVLNode) Hint() hint.Hint {
	return OperationAVLNodeHint
}

func (em *OperationAVLNode) Key() []byte {
	return em.key
}

func (em *OperationAVLNode) Height() int16 {
	return em.height
}

func (em *OperationAVLNode) SetHeight(height int16) error {
	if height < 0 {
		return xerrors.Errorf("height must be greater than zero; height=%d", height)
	}

	em.height = height

	return nil
}

func (em *OperationAVLNode) Left() avl.MutableNode {
	return em.left
}

func (em *OperationAVLNode) LeftKey() []byte {
	if em.left == nil {
		return nil
	}

	return em.left.Key()
}

func (em *OperationAVLNode) SetLeft(node avl.MutableNode) error {
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

func (em *OperationAVLNode) Right() avl.MutableNode {
	return em.right
}

func (em *OperationAVLNode) RightKey() []byte {
	if em.right == nil {
		return nil
	}

	return em.right.Key()
}

func (em *OperationAVLNode) SetRight(node avl.MutableNode) error {
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

func (em *OperationAVLNode) Merge(avl.MutableNode) error {
	return nil
}

func (em *OperationAVLNode) Hash() []byte {
	return em.h
}

func (em *OperationAVLNode) SetHash(h []byte) error {
	em.h = h

	return nil
}

func (em *OperationAVLNode) ResetHash() {
	em.h = nil
}

func (em *OperationAVLNode) LeftHash() []byte {
	if em.left == nil {
		return nil
	}

	return em.left.Hash()
}

func (em *OperationAVLNode) RightHash() []byte {
	if em.right == nil {
		return nil
	}

	return em.right.Hash()
}

func (em *OperationAVLNode) ValueHash() []byte {
	return em.op.Bytes()
}

func (em *OperationAVLNode) Operation() valuehash.Hash {
	return em.op
}
