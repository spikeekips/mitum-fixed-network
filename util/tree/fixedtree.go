package tree

import (
	"bytes"
	"crypto/sha256"
	"math"
	"sync"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/errors"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"golang.org/x/xerrors"
)

var (
	BaseFixedTreeNodeType = hint.MustNewType(0x01, 0x55, "base-fixedtree-node")
	BaseFixedTreeNodeHint = hint.MustHint(BaseFixedTreeNodeType, "0.0.1")
	FixedTreeType         = hint.MustNewType(0x01, 0x56, "fixedtree")
	FixedTreeHint         = hint.MustHint(FixedTreeType, "0.0.1")
)

var (
	InvalidNodeError     = errors.NewError("invalid node")
	NoParentError        = errors.NewError("no node parent")
	EmptyNodeInTreeError = errors.NewError("empty node found in tree")
	EmptyKeyError        = errors.NewError("empty node key")
	EmptyHashError       = errors.NewError("empty node hash")
	NoChildrenError      = errors.NewError("no children")
	HashNotMatchError    = errors.NewError("hash not match")
	InvalidProofError    = errors.NewError("invalid proof")
)

type FixedTreeNode interface {
	hint.Hinter
	isvalid.IsValider
	Index() uint64
	Key() []byte
	Hash() []byte
	SetHash([]byte) FixedTreeNode
	Equal(FixedTreeNode) bool
}

type BaseFixedTreeNode struct {
	index uint64
	key   []byte
	hash  []byte
}

func NewBaseFixedTreeNode(index uint64, key []byte) BaseFixedTreeNode {
	return BaseFixedTreeNode{index: index, key: key}
}

func NewBaseFixedTreeNodeWithHash(index uint64, key, hash []byte) BaseFixedTreeNode {
	return BaseFixedTreeNode{index: index, key: key, hash: hash}
}

func (no BaseFixedTreeNode) Hint() hint.Hint {
	return BaseFixedTreeNodeHint
}

func (no BaseFixedTreeNode) IsValid([]byte) error {
	if len(no.key) < 1 {
		return EmptyKeyError
	}
	if len(no.hash) < 1 {
		return EmptyHashError
	}

	return nil
}

func (no BaseFixedTreeNode) Equal(n FixedTreeNode) bool {
	if no.index != n.Index() {
		return false
	}
	if !bytes.Equal(no.key, n.Key()) {
		return false
	}
	if !bytes.Equal(no.hash, n.Hash()) {
		return false
	}

	return true
}

func (no BaseFixedTreeNode) Index() uint64 {
	return no.index
}

func (no BaseFixedTreeNode) Key() []byte {
	return no.key
}

func (no BaseFixedTreeNode) Hash() []byte {
	return no.hash
}

func (no BaseFixedTreeNode) SetHash(h []byte) FixedTreeNode {
	no.hash = h

	return no
}

type FixedTree struct {
	nodes []FixedTreeNode
}

func NewFixedTree(nodes []FixedTreeNode) FixedTree {
	return FixedTree{nodes: nodes}
}

func (tr FixedTree) Hint() hint.Hint {
	return FixedTreeHint
}

func (tr FixedTree) IsValid([]byte) error {
	if tr.Len() < 1 {
		return nil
	}

	for i := range tr.nodes {
		n := tr.nodes[i]
		if err := n.IsValid(nil); err != nil {
			return err
		} else if int(n.Index()) != i {
			return InvalidNodeError.Errorf("wrong index; %d != %d", n.Index(), i)
		}
	}

	for i := range tr.nodes {
		n := tr.nodes[i]
		if h, err := tr.generateNodeHash(n); err != nil {
			return err
		} else if !bytes.Equal(n.Hash(), h) {
			return InvalidNodeError.Errorf("invalid node hash")
		}
	}

	return nil
}

func (tr FixedTree) Len() int {
	return len(tr.nodes)
}

// Root returns hash of top node
func (tr FixedTree) Root() []byte {
	if tr.Len() < 1 {
		return nil
	}

	return tr.nodes[0].Hash()
}

func (tr FixedTree) Node(index uint64) (FixedTreeNode, error) {
	if int(index) >= tr.Len() {
		return nil, util.NotFoundError.Errorf("node, %d not found", index)
	}

	return tr.nodes[index], nil
}

func (tr FixedTree) Traverse(f func(FixedTreeNode) (bool, error)) error {
	for i := range tr.nodes {
		if keep, err := f(tr.nodes[i]); err != nil {
			return err
		} else if !keep {
			return nil
		}
	}

	return nil
}

// Proof returns the nodes to prove whether node is in tree. It always returns
// root node + N(2 children).
func (tr FixedTree) Proof(index uint64) ([]FixedTreeNode, error) {
	var self FixedTreeNode
	if i, err := tr.Node(index); err != nil {
		return nil, err
	} else {
		self = i
	}

	if tr.Len() < 1 {
		return nil, nil
	}

	var height uint64
	if i, err := tr.height(index); err != nil {
		return nil, err
	} else {
		height = i
	}

	parents := make([]FixedTreeNode, height+1)
	parents[0] = self

	var l uint64 = index
	var i int
	for {
		if j, err := tr.parent(l); err != nil {
			if xerrors.Is(err, NoParentError) {
				break
			}

			return nil, err
		} else {
			parents[i+1] = j
			l = j.Index()
			i++
		}
	}

	pr := make([]FixedTreeNode, (height+1)*2+1)
	for i := range parents {
		n := parents[i]
		if cs, err := tr.children(n.Index()); err != nil {
			if !xerrors.Is(err, NoChildrenError) {
				return nil, err
			}
		} else {
			pr[(i * 2)] = cs[0]
			pr[(i*2)+1] = cs[1]
		}
	}
	pr[len(pr)-1] = tr.nodes[0]

	return pr, nil
}

func (tr FixedTree) children(index uint64) ([]FixedTreeNode, error) {
	if i, err := childrenFixedTree(tr.Len(), index); err != nil {
		return nil, err
	} else {
		if i[1] == 0 {
			return []FixedTreeNode{tr.nodes[i[0]], nil}, nil
		} else {
			return []FixedTreeNode{tr.nodes[i[0]], tr.nodes[i[1]]}, nil
		}
	}
}

func (tr FixedTree) height(index uint64) (uint64, error) {
	return heightFixedTree(tr.Len(), index)
}

func (tr FixedTree) parent(index uint64) (FixedTreeNode, error) {
	var n FixedTreeNode
	if i, err := parentFixedTree(tr.Len(), index); err != nil {
		return n, err
	} else {
		return tr.Node(i)
	}
}

// generateNodeHash generates node hash. Hash was derived from index and key.
func (tr FixedTree) generateNodeHash(n FixedTreeNode) ([]byte, error) {
	if n == nil || len(n.Key()) < 1 {
		return nil, EmptyKeyError
	}

	var left, right FixedTreeNode
	if i, err := tr.children(n.Index()); err != nil {
		if !xerrors.Is(err, NoChildrenError) {
			return nil, err
		}
	} else {
		left = i[0]
		right = i[1]
	}

	return FixedTreeNodeHash(n, left, right)
}

type FixedTreeGenerator struct {
	sync.RWMutex
	FixedTree
	size uint64
}

func NewFixedTreeGenerator(size uint64) *FixedTreeGenerator {
	return &FixedTreeGenerator{
		FixedTree: NewFixedTree(make([]FixedTreeNode, size)),
		size:      size,
	}
}

func (tr *FixedTreeGenerator) Add(n FixedTreeNode) error {
	tr.Lock()
	defer tr.Unlock()

	if len(n.Key()) < 1 {
		return EmptyKeyError
	}

	if n.Index() >= tr.size {
		return xerrors.Errorf("out of range; %d >= %d", n.Index(), tr.size)
	}

	tr.nodes[n.Index()] = n.SetHash(nil)

	return nil
}

func (tr *FixedTreeGenerator) Tree() (FixedTree, error) {
	tr.RLock()
	defer tr.RUnlock()

	if tr.size < 1 {
		return NewFixedTree(tr.nodes), nil
	} else {
		for i := range tr.nodes {
			if tr.nodes[i] == nil {
				return FixedTree{}, EmptyNodeInTreeError.Errorf("node, %d", i)
			}
		}
	}

	if tr.size > 0 && len(tr.nodes[0].Hash()) < 1 {
		for i := range tr.nodes {
			n := tr.nodes[len(tr.nodes)-i-1]
			if h, err := tr.generateNodeHash(n); err != nil {
				return FixedTree{}, err
			} else {
				tr.nodes[n.Index()] = n.SetHash(h)
			}
		}
	}

	return NewFixedTree(tr.nodes), nil
}

func FixedTreeNodeHash(
	self, // self node
	left, // left child
	right FixedTreeNode, // right child
) ([]byte, error) {
	if len(self.Key()) < 1 {
		return nil, EmptyKeyError
	}

	bi := util.Uint64ToBytes(self.Index())
	a := make([]byte, len(self.Key())+len(bi))
	copy(a, bi)
	copy(a[len(bi):], self.Key())

	var lh, rh []byte
	if left != nil {
		lh = left.Hash()
	}
	if right != nil {
		rh = right.Hash()
	}

	return hashNode(util.ConcatBytesSlice(a, lh, rh)), nil
}

func ProveFixedTreeProof(pr []FixedTreeNode) error {
	if err := proveFixedTreeProof(pr); err != nil {
		return InvalidProofError.Wrap(err)
	} else {
		return nil
	}
}

func proveFixedTreeProof(pr []FixedTreeNode) error {
	switch n := len(pr); {
	case n < 1:
		return xerrors.Errorf("nothing to prove")
	case n%2 != 1:
		return xerrors.Errorf("invalid proof; len=%d", n)
	case pr[len(pr)-1].Index() != 0:
		return xerrors.Errorf("root node not found")
	}

	for i := range pr {
		if err := pr[i].IsValid(nil); err != nil {
			return InvalidNodeError.Errorf("node, %d", i)
		}
	}

	for i := 0; i < len(pr[:len(pr)-1])/2; i++ {
		a, b := pr[(i*2)], pr[(i*2)+1]
		if p, err := parentNodeInProof(i, pr, a.Index()); err != nil {
			return xerrors.Errorf("nodes, %d and %d: %w", a.Index(), b.Index(), err)
		} else if h, err := FixedTreeNodeHash(p, a, b); err != nil {
			return err
		} else if !bytes.Equal(p.Hash(), h) {
			return HashNotMatchError.Errorf("node, %d has wrong hash", p.Index())
		}
	}

	return nil
}

func parentNodeInProof(i int, pr []FixedTreeNode, index uint64) (FixedTreeNode, error) {
	maxSize := int(math.Pow(2, float64(len(pr[:len(pr)-1])/2)+1)) - 1

	var p FixedTreeNode
	switch j, err := parentFixedTree(maxSize, index); {
	case err != nil:
		return p, err
	case i < (len(pr[:len(pr)-1])/2)-1:
		pa, pb := pr[(i*2)+2], pr[(i*2)+2+1]
		if j == pa.Index() {
			p = pa
		} else {
			p = pb
		}
	default:
		p = pr[len(pr)-1]
	}

	if len(p.Key()) < 1 {
		return p, xerrors.Errorf("parent node not found")
	}

	return p, nil
}

func heightFixedTree(size int, index uint64) (uint64, error) {
	if int(index) >= size {
		return 0, util.NotFoundError.Errorf("node, %d not found", index)
	} else if index == 0 {
		return 0, nil
	}

	return uint64(math.Log(float64(index+1)) / math.Log(2)), nil
}

func parentFixedTree(size int, index uint64) (uint64, error) {
	var height uint64
	switch i, err := heightFixedTree(size, index); {
	case err != nil:
		return 0, err
	case i == 0:
		return 0, NoParentError
	default:
		height = i
	}

	currentFirst := uint64(math.Pow(2, float64(height)) - 1)
	pos := index - currentFirst

	if pos%2 == 1 {
		pos--
	}

	upFirst := uint64(math.Pow(2, float64(height-1)) - 1)
	return upFirst + pos/2, nil
}

func childrenFixedTree(size int, index uint64) ([]uint64, error) {
	var height uint64
	if i, err := heightFixedTree(size, index); err != nil {
		return nil, err
	} else {
		height = i
	}

	currentFirst := uint64(math.Pow(2, float64(height)) - 1)
	pos := index - currentFirst
	nextFirst := uint64(math.Pow(2, float64(height+1)) - 1)

	children := make([]uint64, 2)
	if i := nextFirst + pos*2; i >= uint64(size) {
		return nil, NoChildrenError
	} else {
		children[0] = i
	}

	if i := nextFirst + pos*2 + 1; i < uint64(size) {
		children[1] = i
	}

	return children, nil
}

func hashNode(b []byte) []byte {
	h := sha256.Sum256(b)

	return h[:]
}
