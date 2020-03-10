package state

import (
	"github.com/spikeekips/avl"
	avlHashable "github.com/spikeekips/avl/hashable"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
)

var StateV0AVLNodeHint hint.Hint = hint.MustHint(hint.Type{0x10, 0x02}, "0.0.1")

type StateV0AVLNode struct {
	height int16
	left   avlHashable.HashableMutableNode
	right  avlHashable.HashableMutableNode
	h      []byte
	state  *StateV0 // State
}

func NewStateV0AVLNode(st *StateV0) *StateV0AVLNode {
	return &StateV0AVLNode{state: st}
}

func (stav StateV0AVLNode) Hint() hint.Hint {
	return StateV0AVLNodeHint
}

func (stav *StateV0AVLNode) Key() []byte {
	return []byte(stav.state.Key())
}

func (stav *StateV0AVLNode) Height() int16 {
	return stav.height
}

func (stav *StateV0AVLNode) SetHeight(height int16) error {
	if height < 0 {
		return xerrors.Errorf("height must be greater than zero; height=%d", height)
	}

	stav.height = height

	return nil
}

func (stav *StateV0AVLNode) Left() avl.MutableNode {
	return stav.left
}

func (stav *StateV0AVLNode) LeftKey() []byte {
	if stav.left == nil {
		return nil
	}

	return stav.left.Key()
}

func (stav *StateV0AVLNode) SetLeft(node avl.MutableNode) error {
	if node == nil {
		stav.left = nil
		return nil
	}

	if avl.EqualKey(stav.Key(), node.Key()) {
		return xerrors.Errorf("left is same node; key=%v", stav.Key())
	}

	m, ok := node.(avlHashable.HashableMutableNode)
	if !ok {
		return xerrors.Errorf("not HashableMutableNode; %T", node)
	}

	stav.left = m

	return nil
}

func (stav *StateV0AVLNode) Right() avl.MutableNode {
	return stav.right
}

func (stav *StateV0AVLNode) RightKey() []byte {
	if stav.right == nil {
		return nil
	}

	return stav.right.Key()
}

func (stav *StateV0AVLNode) SetRight(node avl.MutableNode) error {
	if node == nil {
		stav.right = nil
		return nil
	}

	if avl.EqualKey(stav.Key(), node.Key()) {
		return xerrors.Errorf("right is same node; key=%v", stav.Key())
	}

	m, ok := node.(avlHashable.HashableMutableNode)
	if !ok {
		return xerrors.Errorf("not HashableMutableNode; %T", node)
	}

	stav.right = m

	return nil
}

func (stav *StateV0AVLNode) Merge(node avl.MutableNode) error {
	var m *StateV0AVLNode
	if hinter, ok := node.(hint.Hinter); !ok {
		return xerrors.Errorf("not hinter: %T", node)
	} else if err := stav.Hint().IsCompatible(hinter.Hint()); err != nil {
		return err
	} else if n, ok := node.(*StateV0AVLNode); !ok {
		return xerrors.Errorf("not *StateV0AVLNode: %T", node)
	} else {
		m = n
	}

	stav.state = m.state

	return nil
}

func (stav *StateV0AVLNode) Hash() []byte {
	return stav.h
}

func (stav *StateV0AVLNode) SetHash(h []byte) error {
	stav.h = h

	return nil
}

func (stav *StateV0AVLNode) ResetHash() {
	stav.h = nil
}

func (stav *StateV0AVLNode) LeftHash() []byte {
	if stav.left == nil {
		return nil
	}

	return stav.left.Hash()
}

func (stav *StateV0AVLNode) RightHash() []byte {
	if stav.right == nil {
		return nil
	}

	return stav.right.Hash()
}

func (stav *StateV0AVLNode) ValueHash() []byte {
	return stav.state.Hash().Bytes()
}

func (stav *StateV0AVLNode) State() State {
	return stav.state
}
