package tree

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/sha3"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
)

type testFixedTree struct {
	suite.Suite
}

func (t *testFixedTree) TestNew() {
	ft := NewFixedTreeGenerator(10, nil)
	t.NotNil(ft)
	t.Equal(30, len(ft.nodes))

	ft = NewFixedTreeGenerator(9, nil)
	t.NotNil(ft)
	t.Equal(27, len(ft.nodes))

	tr, err := ft.Tree()
	t.NoError(err)

	t.Implements((*hint.Hinter)(nil), tr)
}

func (t *testFixedTree) TestIndex() {
	{
		ft := NewFixedTreeGenerator(3, nil)
		t.NotNil(ft)

		t.NoError(ft.Append(nil, nil))
		t.NoError(ft.Append(nil, nil))
		t.NoError(ft.Append(nil, nil))
		err := ft.Append(nil, nil)
		t.Contains(err.Error(), "already filled")
	}

	{
		ft := NewFixedTreeGenerator(3, nil)
		t.NotNil(ft)

		t.NoError(ft.Add(0, nil, nil))
		t.NoError(ft.Add(1, nil, nil))
		t.NoError(ft.Add(2, nil, nil))
		err := ft.Add(3, nil, nil)
		t.Contains(err.Error(), "over size")
	}
}

func (t *testFixedTree) TestHeight() {
	ft := NewFixedTreeGenerator(20, nil)
	t.NotNil(ft)

	t.Equal(0, ft.height(0))
	t.Equal(-1, ft.height(20))
}

func (t *testFixedTree) TestChildren() {
	ft := NewFixedTreeGenerator(20, nil)

	for i := 0; i < ft.size; i++ {
		t.NoError(ft.Add(i, []byte(fmt.Sprintf("%d", i)), nil))
	}

	for i := 0; i < ft.size; i++ {
		children := ft.children(i)

		t.T().Log(fmt.Sprintf("index=%d childrena=%v", i, children))
		if children[0] >= 0 {
			t.NotNil(ft.nodes[children[0]*3])
		}
		if children[1] >= 0 {
			t.NotNil(ft.nodes[children[1]*3])
		}
	}
}

func (t *testFixedTree) TestNodeHash() {
	ft := NewFixedTreeGenerator(20, func(b []byte) []byte {
		h := sha3.Sum256(b)
		return h[:]
	})

	for i := 0; i < ft.size; i++ {
		b := []byte(fmt.Sprintf("%d", i))
		t.NoError(ft.Add(i, b, nil))
		t.Equal(b, ft.Key(i))
	}

	expected_hashes := []string{
		"87d7dd884b8252db03c5d5a8b0d95a0ff503ed828b33122d9109cef1c102169d",
		"bed71740335e68b67278fb37b61a03d30493d16535534f068cc039613e9eca91",
		"b7db7bc7d20c5fdabf4fe7e0ddd8923f2fd023613898a20f0b48b3fbb27c8770",
		"61eacd1f797014c82025a8a10d0a8ac15bcce6cc4662f361fab2787b98962ed2",
		"3e8c7ccad22bab1ba595e67a8b52aa67021f7fd508c87cc91e8b44acb2031bcd",
		"9435ec791ae00ada11514164db866a7bf93a20d52104bddb69e3716d7a7a232a",
		"a6382727b782336163a0c20aaa3cefe78413744fe30fff1cfa405e2bf65d79a9",
		"59166b96e2712afec64e6652243e2c44d8e33291b270d5e6c3b302527bd4ec2f",
		"fae08a87fc3e4a056fad35df71f74e7419bd97a46f902b6bace4e9f814b4173a",
		"831c108325897b2f690f0d5a2d7681741c72c4004f7fe936dbd8814cb509bdc1",
		"dd121e36961a04627eacff629765dd3528471ed745c1e32222db4a8a5f3421c4",
		"4410fc15c5a3cde7a2b5366a41dbc95e6547a6021efdff98cfcd5e875e8c3c70",
		"1a9a118cb653759c3fcb3bd5060e6f9910c8c27008dd11fe4315f4635c9caa98",
		"1ad7a51ebb6db8cfd0f40d83e398f0a8ad6e7fd4b98e6623b92cfc7c18c4325a",
		"1f5272c162bddcec544967f3c32b238b0f632d365fe95c6fb0929db8cbf2282c",
		"71f0c2511c6d5dae680e288d7d627eb127f3b3cc1079f0fc497170c4b35759f7",
		"958b08cb3a6f8252890b89292372d10357890e39ca35cbc684d3ecd9e4f052a6",
		"c8b6a189ddbf2b1dd605d19a9889d4cd1bdb5a451e614a02f83fb8be46dde633",
		"a5c7cd33a255de5992d6d74b34a3ebdde7d1e922de25dac1d30ea3f0ad88df19",
		"7548240b8da85518ebb5dfa9e45899b43c64cb867a6f21f90384d283ef142cae",
	}

	_ = ft.Hash(0)
	for i := 0; i < ft.size; i++ {
		t.Equal(expected_hashes[i], hex.EncodeToString(ft.Hash(i)))
	}

	t.Equal(ft.size*3, len(ft.Nodes()))
}

func (t *testFixedTree) TestNilKey() {
	hashFunc := func(b []byte) []byte {
		h := sha3.Sum256(b)

		return h[:]
	}
	ft := NewFixedTreeGenerator(200, hashFunc)

	for i := 0; i < ft.size; i++ {
		var k []byte
		if i == 10 {
			k = []byte("9d8431a2-e16d-4723-b495-1739e26f5f7e")
		}
		t.NoError(ft.Add(i, k, nil))
	}

	expected := make([]string, ft.size)
	for i := 0; i < ft.size; i++ {
		expected[i] = ""
	}

	expected[0] = "c4c58dec3e69a44be39c92bacee66a2bd9cda0c58e757001418f3c6321b8b1d7"
	expected[1] = "7444ba93c9ad43ef0960759d209dddb963956de72fb47cdf515c851801d01878"
	expected[4] = "2eff19b70aa472dca7c689d3638be2b2baf9c7e2f13f2ccd4a3f731b64715bf8"
	expected[10] = hex.EncodeToString(hashFunc(ft.Key(10)))

	_ = ft.Hash(0)
	for i := 0; i < ft.size; i++ {
		t.Equal(expected[i], hex.EncodeToString(ft.nodes[(i*3)+1]), "index=%d", i)
	}

	t.Equal(ft.size*3, len(ft.Nodes()))
}

func (t *testFixedTree) TestAppend() {
	var size uint = 200000
	var root []byte
	{
		ft := NewFixedTreeGenerator(size, nil)

		s := time.Now()
		for i := 0; i < ft.size; i++ {
			t.NoError(ft.Add(i, []byte(fmt.Sprintf("%d", i)), nil))
		}
		t.T().Log("from root:  insert: elapsed", ft.size, time.Since(s))

		s = time.Now()
		root = ft.Hash(0)
		t.T().Log("from root: hashing: elapsed", ft.size, time.Since(s))
	}

	{
		ft := NewFixedTreeGenerator(size, nil)

		s := time.Now()
		for i := ft.size - 1; i >= 0; i-- {
			t.NoError(ft.Append([]byte(fmt.Sprintf("%d", i)), nil))
		}
		t.T().Log(" from end:  insert: elapsed", ft.size, time.Since(s))

		s = time.Now()
		root0 := ft.Hash(0)
		t.T().Log(" from end: hashing: elapsed", ft.size, time.Since(s))

		t.Equal(root, root0)
	}
}

func (t *testFixedTree) TestParallel() {
	var size uint = 200000

	var root []byte
	{
		ft := NewFixedTreeGenerator(size, nil)

		s := time.Now()
		for i := 0; i < ft.size; i++ {
			t.NoError(ft.Add(i, []byte(fmt.Sprintf("%d", i)), nil))
		}
		t.T().Log("     add:  insert: elapsed", ft.size, time.Since(s))

		s = time.Now()
		root = ft.Hash(0)
		t.T().Log("     add: hashing: elapsed", ft.size, time.Since(s))
	}

	{
		l := make([]int, size)
		for i := 0; i < int(size); i++ {
			l[i] = i
		}

		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(l), func(i, j int) { l[i], l[j] = l[j], l[i] })

		ft := NewFixedTreeGenerator(size, nil)

		indexChan := make(chan int, size)
		done := make(chan struct{}, size)
		s := time.Now()

		for i := 0; i < 10; i++ {
			go func() {
				for i := range indexChan {
					t.NoError(ft.Add(i, []byte(fmt.Sprintf("%d", i)), nil))
					done <- struct{}{}
				}
			}()
		}

		go func() {
			for _, i := range l {
				indexChan <- i
			}
			close(indexChan)
		}()

		var count uint

	end:
		for _ = range done {
			count++
			if count >= size {
				break end
			}
		}

		t.T().Log("parallel:  insert: elapsed", ft.size, time.Since(s))

		s = time.Now()
		root0 := ft.Hash(0)
		t.T().Log("parallel: hashing: elapsed", ft.size, time.Since(s))

		t.Equal(root, root0)
	}
}

func (t *testFixedTree) TestProof() {
	ft := NewFixedTreeGenerator(20, func(b []byte) []byte {
		h := sha3.Sum256(b)
		return h[:]
	})

	for i := 0; i < ft.size; i++ {
		k := fmt.Sprintf("%d", i)
		t.NoError(ft.Add(i, []byte(k), nil))
	}

	_ = ft.Hash(0)

	pr, err := ft.Proof(20)
	t.Contains(err.Error(), "over size")

	pr, err = ft.Proof(19)
	t.NoError(err)
	t.Equal(22, len(pr))

	var keys []string
	var hashes [][]byte
	for i := 0; i < len(pr)/2; i++ {
		key := pr[i*2]
		keys = append(keys, string(key))

		h := pr[i*2+1]
		hashes = append(hashes, h)
	}

	t.Equal([]string{"19", "9", "9", "10", "4", "3", "4", "1", "1", "2", "0"}, keys)

	ids := []int{19, 9, 9, 10, 4, 3, 4, 1, 1, 2, 0}
	for i, h := range hashes {
		t.Equal(ft.Hash(ids[i]), h)
	}
}

func (t *testFixedTree) TestProve() {
	{ // empty proof
		err := ProveFixedTreeProof(nil, nil)
		t.Contains(err.Error(), "nothing to prove")
	}

	{ // wrong sized proof
		err := ProveFixedTreeProof([][]byte{nil}, nil)
		t.Contains(err.Error(), "invalid proof")
	}

	ft := NewFixedTreeGenerator(20, nil)

	for i := 0; i < ft.size; i++ {
		t.NoError(ft.Add(i, []byte(fmt.Sprintf("%d", i)), nil))
	}

	_ = ft.Hash(0)

	pr, err := ft.Proof(20)
	t.Contains(err.Error(), "over size")

	{ // even
		pr, err = ft.Proof(18)
		t.NoError(err)
		t.Equal(24, len(pr))

		err = ProveFixedTreeProof(pr, nil)
		t.NoError(err)
	}

	{ // odd
		pr, err = ft.Proof(19)
		t.NoError(err)
		t.Equal(22, len(pr))

		err = ProveFixedTreeProof(pr, nil)
		t.NoError(err)
	}
}

func (t *testFixedTree) TestProveWrongHashInTheMiddle() {
	ft := NewFixedTreeGenerator(20, nil)

	for i := 0; i < ft.size; i++ {
		t.NoError(ft.Add(i, []byte(fmt.Sprintf("%d", i)), nil))
	}

	_ = ft.Hash(0)

	pr, err := ft.Proof(18)
	t.NoError(err)
	t.Equal(24, len(pr))

	err = ProveFixedTreeProof(pr, nil)
	t.NoError(err)

	pr[13] = []byte("showme")
	err = ProveFixedTreeProof(pr, nil)
	t.Contains(err.Error(), "wrong hash in the middle found; index=16")
}

func (t *testFixedTree) TestValidate() {
	ft := NewFixedTreeGenerator(20, nil)

	for i := 0; i < ft.size; i++ {
		t.NoError(ft.Add(i, []byte(fmt.Sprintf("%d", i)), nil))
	}

	_ = ft.Hash(0)

	fv, err := NewFixedTree(ft.Nodes(), nil)
	t.NoError(err)
	t.NoError(fv.IsValid(nil))

	{
		nodes := ft.Nodes()

		_, err := NewFixedTree(nodes[:4], nil)
		t.Contains(err.Error(), "invalid nodes")
	}

	{ // wrong hash
		nodes := ft.Nodes()
		nodes[13] = []byte("showme")

		fv, err := NewFixedTree(nodes, nil)
		t.NoError(err)
		err = fv.IsValid(nil)
		t.Contains(err.Error(), "wrong hash; index=4")
	}
}

func (t *testFixedTree) TestTraverse() {
	ft := NewFixedTreeGenerator(19, nil)

	keys := make([][]byte, 19)
	for i := 0; i < ft.size; i++ {
		key := []byte(fmt.Sprintf("%d", i))
		keys[i] = key
		t.NoError(ft.Add(i, key, key))
	}

	tr, err := ft.Tree()
	t.NoError(err)

	t.NoError(tr.Traverse(func(i int, key, h, v []byte) (bool, error) {
		t.Equal(keys[i], key)
		t.Equal(keys[i], v)
		t.Equal(ft.Key(i), key)
		t.Equal(ft.Hash(i), h)
		t.Equal(ft.Extra(i), v)

		return true, nil
	}))
}

func (t *testFixedTree) TestEncode() {
	ft := NewFixedTreeGenerator(20, nil)

	for i := 0; i < ft.size; i++ {
		t.NoError(ft.Add(i, []byte(fmt.Sprintf("%d", i)), nil))
	}

	tr, err := ft.Tree()
	t.NoError(err)

	b, err := jsonenc.Marshal(tr)
	t.NoError(err)
	t.NotNil(b)

	var uft FixedTree
	t.NoError(uft.UnmarshalJSON(b))

	t.Equal(ft.Len(), uft.Len())
	t.True(t.compareBytes(ft.Nodes(), uft.Nodes()))
}

func (t *testFixedTree) compareBytes(a, b [][]byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if len(a[i]) != len(b[i]) {
			return false
		} else if !bytes.Equal(a[i], b[i]) {
			return false
		}
	}

	return true
}

func (t *testFixedTree) TestDump() {
	ft := NewFixedTreeGenerator(20, nil)

	for i := 0; i < ft.size; i++ {
		t.NoError(ft.Add(i, []byte(fmt.Sprintf("%d", i)), nil))
	}

	tr, err := ft.Tree()
	t.NoError(err)
	t.NoError(tr.IsValid(nil))

	var buf bytes.Buffer
	t.NoError(tr.Dump(&buf))

	utr, err := LoadFixedTreeFromReader(bytes.NewReader(buf.Bytes()))
	t.NoError(err)
	t.NoError(utr.IsValid(nil))

	t.Equal(tr.Len(), utr.Len())
	t.Equal(tr.Root(), utr.Root())
}

func TestFixedTree(t *testing.T) {
	suite.Run(t, new(testFixedTree))
}
