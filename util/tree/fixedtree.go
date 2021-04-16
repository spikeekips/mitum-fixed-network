package tree

import (
	"bytes"
	"crypto/sha256"
	"math"
	"sync"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
)

var (
	FixedTreeType = hint.MustNewType(0x01, 0x55, "fixedtree")
	FixedTreeHint = hint.MustHint(FixedTreeType, "0.0.1")
)

var DefaultFixedTreeHashFunc = func(b []byte) []byte {
	h := sha256.Sum256(b)

	return h[:]
}

type baseFixedTree struct {
	size     int
	nodes    [][]byte
	hashFunc func([]byte) []byte
}

func (ft baseFixedTree) Tree() (FixedTree, error) {
	return NewFixedTree(ft.nodes, ft.hashFunc)
}

func (ft baseFixedTree) Root() []byte {
	if ft.size < 1 {
		return nil
	}

	return ft.Hash(0)
}

func (ft baseFixedTree) IsEmpty() bool {
	return ft.size < 1
}

func (ft baseFixedTree) Len() int {
	return ft.size
}

func (ft baseFixedTree) Key(i int) []byte {
	return ft.nodes[i*3]
}

func (ft baseFixedTree) Hash(i int) []byte {
	return ft.nodes[i*3+1]
}

func (ft baseFixedTree) Extra(i int) []byte {
	return ft.nodes[i*3+2]
}

func (ft baseFixedTree) Nodes() [][]byte {
	return ft.nodes
}

func (ft baseFixedTree) Proof(i int) ([][]byte, error) {
	if l := ft.size; i+1 > l {
		return nil, xerrors.Errorf("index, %d over size, %d", i, l)
	}

	h := ft.height(i)
	if h == 0 {
		return [][]byte{ft.nodes[0], ft.nodes[1]}, nil
	}

	var ps []int
	var l int = i
	for {
		p := ft.parent(l)
		if p < 0 {
			break
		}

		ps = append(ps, p)
		l = p
	}

	var pr [][]byte

	insert := func(i int) {
		if i < 0 {
			return
		}

		pr = append(pr, ft.Key(i), ft.Hash(i))
	}

	for i := 0; i < len(ps); i++ {
		j := ps[i]

		ch := ft.children(j)
		insert(ch[0])
		insert(ch[1])
		insert(j)
	}

	return pr, nil
}

func (ft baseFixedTree) height(i int) int {
	if i == 0 {
		return 0
	} else if i < 0 || i >= ft.size {
		return -1
	}

	return int(math.Log(float64(i+1)) / math.Log(2))
}

func (ft baseFixedTree) parent(i int) int {
	h := ft.height(i)
	if h == 0 {
		return -1
	}

	currentFirst := int(math.Pow(2, float64(h)) - 1)
	pos := i - currentFirst

	if pos%2 == 1 {
		pos--
	}

	upFirst := int(math.Pow(2, float64(h-1)) - 1)
	upPos := upFirst + pos/2

	return upPos
}

func (ft baseFixedTree) children(i int) [2]int {
	h := ft.height(i)
	currentFirst := int(math.Pow(2, float64(h)) - 1)
	pos := i - currentFirst
	nextFirst := int(math.Pow(2, float64(h+1)) - 1)

	var children [2]int
	if n := nextFirst + pos*2; n >= ft.size {
		children[0] = -1
	} else {
		children[0] = n
	}
	if n := nextFirst + pos*2 + 1; n >= ft.size {
		children[1] = -1
	} else {
		children[1] = n
	}

	return children
}

func (ft baseFixedTree) generateNodeHash(i int) []byte {
	var b [3][]byte

	key := ft.nodes[i*3]
	extra := ft.nodes[i*3+2]
	if len(key) > 0 {
		b[0] = make([]byte, len(key)+len(extra))
		copy(b[0], key)
		copy(b[0][len(key):], extra)
	}

	ch := ft.children(i)
	if ch[0] >= 0 {
		b[1] = ft.Hash(ch[0])
	}
	if ch[1] >= 0 {
		b[2] = ft.Hash(ch[1])
	}

	return FixedTreeNodeHash(b[0], b[1], b[2], ft.hashFunc)
}

func (ft baseFixedTree) Traverse(f func(
	int, // index
	[]byte, // key
	[]byte, // hash
	[]byte, // extra
) (bool, error)) error {
	for i := 0; i < ft.size; i++ {
		if keep, err := f(i, ft.Key(i), ft.Hash(i), ft.Extra(i)); err != nil {
			return err
		} else if !keep {
			return nil
		}
	}

	return nil
}

type FixedTreeGenerator struct {
	sync.RWMutex
	baseFixedTree
	lastIndex int
}

func NewFixedTreeGenerator(size uint, hashFunc func([]byte) []byte) *FixedTreeGenerator {
	if hashFunc == nil {
		hashFunc = DefaultFixedTreeHashFunc
	}

	return &FixedTreeGenerator{
		baseFixedTree: baseFixedTree{
			size:     int(size),
			nodes:    make([][]byte, size*3),
			hashFunc: hashFunc,
		},
		lastIndex: int(size) - 1,
	}
}

func (ft *FixedTreeGenerator) Tree() (FixedTree, error) {
	if ft.size > 0 {
		_ = ft.hash(0)
	}

	return NewFixedTree(ft.nodes, ft.hashFunc)
}

// Add adds key to tree. extra is not counted to hashing.
func (ft *FixedTreeGenerator) Add(i int, key, extra []byte) error {
	ft.Lock()
	defer ft.Unlock()

	if l := ft.size; i+1 > l {
		return xerrors.Errorf("index, %d over size, %d", i, l)
	}

	ft.nodes[i*3] = key
	ft.nodes[i*3+2] = extra

	return nil
}

func (ft *FixedTreeGenerator) Append(key, extra []byte) error {
	ft.Lock()
	defer ft.Unlock()

	i := ft.lastIndex
	if i < 0 {
		return xerrors.Errorf("already filled")
	}

	ft.nodes[i*3] = key
	ft.nodes[i*3+1] = ft.hash(i)
	ft.nodes[i*3+2] = extra
	ft.lastIndex--

	return nil
}

func (ft *FixedTreeGenerator) Nodes() [][]byte {
	ft.RLock()
	defer ft.RUnlock()

	return ft.baseFixedTree.Nodes()
}

func (ft *FixedTreeGenerator) Key(i int) []byte {
	ft.RLock()
	defer ft.RUnlock()

	return ft.baseFixedTree.Key(i)
}

func (ft *FixedTreeGenerator) Hash(i int) []byte {
	ft.RLock()
	defer ft.RUnlock()

	return ft.hash(i)
}

func (ft *FixedTreeGenerator) Proof(i int) ([][]byte, error) {
	ft.RLock()
	defer ft.RUnlock()

	return ft.baseFixedTree.Proof(i)
}

func (ft *FixedTreeGenerator) hash(i int) []byte {
	if n := ft.nodes[(i*3)+1]; n != nil {
		return n
	}

	ch := ft.children(i)
	if ch[0] >= 0 && ft.baseFixedTree.Hash(ch[0]) == nil {
		_ = ft.hash(ch[0])
	}
	if ch[1] >= 0 && ft.baseFixedTree.Hash(ch[1]) == nil {
		_ = ft.hash(ch[1])
	}

	ft.nodes[(i*3)+1] = ft.generateNodeHash(i)

	return ft.nodes[(i*3)+1]
}

type FixedTree struct {
	baseFixedTree
}

func NewFixedTree(nodes [][]byte, hashFunc func([]byte) []byte) (FixedTree, error) {
	if n := len(nodes); n%3 != 0 {
		return FixedTree{}, xerrors.Errorf("invalid nodes; len=%d", n)
	}

	if hashFunc == nil {
		hashFunc = DefaultFixedTreeHashFunc
	}

	return FixedTree{
		baseFixedTree: baseFixedTree{
			size:     len(nodes) / 3,
			nodes:    nodes,
			hashFunc: hashFunc,
		},
	}, nil
}

func (ft FixedTree) Hint() hint.Hint {
	return FixedTreeHint
}

func (ft FixedTree) IsValid([]byte) error {
	if ft.size < 1 {
		return nil
	}

	for i := ft.size - 1; i >= 0; i-- {
		if !bytes.Equal(ft.nodes[i*3+1], ft.generateNodeHash(i)) {
			return xerrors.Errorf("wrong hash; index=%d", i)
		}
	}

	return nil
}

func FixedTreeNodeHash(
	a, // key
	b, // left child
	c []byte, // right child
	hashFunc func([]byte) []byte,
) []byte {
	if len(a) < 1 && len(b) < 1 && len(c) < 1 {
		return nil
	}

	return hashFunc(util.ConcatBytesSlice(a, b, c))
}

func ProveFixedTreeProof(pr [][]byte, hashFunc func([]byte) []byte) error {
	n := len(pr)
	if n < 1 {
		return xerrors.Errorf("nothing to prove")
	} else if n%2 != 0 {
		return xerrors.Errorf("invalid proof; len=%d", n)
	} else if i := n % 3; i != 0 && i != 1 {
		return xerrors.Errorf("invalid proof; len=%d", n)
	}

	if hashFunc == nil {
		hashFunc = DefaultFixedTreeHashFunc
	}

	pos := (((n % 3) + 1) % 2) + 1
	if n/2 > pos { // NOTE check head nodes
		for i := 0; i < pos; i++ {
			h := FixedTreeNodeHash(pr[i*2], nil, nil, hashFunc)
			if !bytes.Equal(pr[i*2+1], h) {
				return xerrors.Errorf("wrong hash found; index=%d", pos*2)
			}
		}
	}

	var prev []byte
	for {
		var key, lh, rh, h []byte
		if pos < 2 {
			l := pr[0:4]
			key = l[2]
			lh = l[1]
			h = l[3]
		} else {
			l := pr[(pos-2)*2 : pos*2+2]
			key = l[4]
			lh = l[1]
			rh = l[3]
			h = l[5]
		}

		if prev != nil {
			if (lh != nil && !bytes.Equal(lh, prev)) && (rh != nil && !bytes.Equal(rh, prev)) {
				return xerrors.Errorf("wrong hash in the middle found; index=%d", pos*2)
			}
		}

		nh := FixedTreeNodeHash(key, lh, rh, hashFunc)
		if !bytes.Equal(h, nh) {
			return xerrors.Errorf("wrong hash found; index=%d", pos*2+2)
		}

		prev = nh

		if pos+1 >= len(pr)/2 {
			break
		}

		pos += 3
	}

	return nil
}
