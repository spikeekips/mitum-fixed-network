package state

import (
	"github.com/spikeekips/avl"
	avlHashable "github.com/spikeekips/avl/hashable"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/tree"
	"github.com/spikeekips/mitum/util/hint"
)

var (
	StateV0AVLNodeType        = hint.MustNewType(0x01, 0x62, "state-v0-avlnode")
	StateV0AVLNodeHint        = hint.MustHint(StateV0AVLNodeType, "0.0.1")
	StateV0AVLNodeMutableType = hint.MustNewType(0x01, 0x63, "state-v0-avlnode-mutable")
	StateV0AVLNodeMutableHint = hint.MustHint(StateV0AVLNodeMutableType, "0.0.1")
)

type StateV0AVLNode struct {
	h         []byte
	height    int16
	left      []byte
	leftHash  []byte
	right     []byte
	rightHash []byte
	state     *StateV0
}

func NewStateV0AVLNode(st *StateV0AVLNodeMutable) StateV0AVLNode {
	return StateV0AVLNode{
		height:    st.height,
		left:      st.LeftKey(),
		leftHash:  st.LeftHash(),
		right:     st.RightKey(),
		rightHash: st.RightHash(),
		h:         st.Hash(),
		state:     st.state,
	}
}

func (stav StateV0AVLNode) Hint() hint.Hint {
	return StateV0AVLNodeHint
}

func (stav StateV0AVLNode) Key() []byte {
	return []byte(stav.state.Key())
}

func (stav StateV0AVLNode) Height() int16 {
	return stav.height
}

func (stav StateV0AVLNode) Hash() []byte {
	return stav.h
}

func (stav StateV0AVLNode) LeftKey() []byte {
	return stav.left
}

func (stav StateV0AVLNode) RightKey() []byte {
	return stav.right
}

func (stav StateV0AVLNode) LeftHash() []byte {
	return stav.leftHash
}

func (stav StateV0AVLNode) RightHash() []byte {
	return stav.rightHash
}

func (stav StateV0AVLNode) ValueHash() []byte {
	return stav.state.Hash().Bytes()
}

func (stav StateV0AVLNode) State() State {
	return stav.state
}

func (stav StateV0AVLNode) Immutable() tree.Node {
	return stav
}

type StateV0AVLNodeMutable struct {
	height int16
	left   avlHashable.HashableMutableNode
	right  avlHashable.HashableMutableNode
	h      []byte
	state  *StateV0 // State
}

func NewStateV0AVLNodeMutable(st *StateV0) *StateV0AVLNodeMutable {
	return &StateV0AVLNodeMutable{state: st}
}

func (stav StateV0AVLNodeMutable) Hint() hint.Hint {
	return StateV0AVLNodeMutableHint
}

func (stav *StateV0AVLNodeMutable) Key() []byte {
	return []byte(stav.state.Key())
}

func (stav *StateV0AVLNodeMutable) Height() int16 {
	return stav.height
}

func (stav *StateV0AVLNodeMutable) SetHeight(height int16) error {
	if height < 0 {
		return xerrors.Errorf("height must be greater than zero; height=%d", height)
	}

	stav.height = height

	return nil
}

func (stav *StateV0AVLNodeMutable) Left() avl.MutableNode {
	return stav.left
}

func (stav *StateV0AVLNodeMutable) LeftKey() []byte {
	if stav.left == nil {
		return nil
	}

	return stav.left.Key()
}

func (stav *StateV0AVLNodeMutable) SetLeft(node avl.MutableNode) error {
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

func (stav *StateV0AVLNodeMutable) Right() avl.MutableNode {
	return stav.right
}

func (stav *StateV0AVLNodeMutable) RightKey() []byte {
	if stav.right == nil {
		return nil
	}

	return stav.right.Key()
}

func (stav *StateV0AVLNodeMutable) SetRight(node avl.MutableNode) error {
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

func (stav *StateV0AVLNodeMutable) Merge(node avl.MutableNode) error {
	var m *StateV0AVLNodeMutable
	if hinter, ok := node.(hint.Hinter); !ok {
		return xerrors.Errorf("not hinter: %T", node)
	} else if err := stav.Hint().IsCompatible(hinter.Hint()); err != nil {
		return err
	} else if n, ok := node.(*StateV0AVLNodeMutable); !ok {
		return xerrors.Errorf("not *StateV0AVLNodeMutable: %T", node)
	} else {
		m = n
	}

	stav.state = m.state

	return nil
}

func (stav *StateV0AVLNodeMutable) Hash() []byte {
	return stav.h
}

func (stav *StateV0AVLNodeMutable) SetHash(h []byte) error {
	stav.h = h

	return nil
}

func (stav *StateV0AVLNodeMutable) ResetHash() {
	stav.h = nil
}

func (stav *StateV0AVLNodeMutable) LeftHash() []byte {
	if stav.left == nil {
		return nil
	}

	return stav.left.Hash()
}

func (stav *StateV0AVLNodeMutable) RightHash() []byte {
	if stav.right == nil {
		return nil
	}

	return stav.right.Hash()
}

func (stav *StateV0AVLNodeMutable) ValueHash() []byte {
	return stav.state.Hash().Bytes()
}

func (stav *StateV0AVLNodeMutable) State() State {
	return stav.state
}

func (stav *StateV0AVLNodeMutable) Immutable() tree.Node {
	return NewStateV0AVLNode(stav)
}
