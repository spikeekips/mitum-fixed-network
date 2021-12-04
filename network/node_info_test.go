//go:build test
// +build test

package network

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testNodeInfo struct {
	suite.Suite
	encs    *encoder.Encoders
	encJSON encoder.Encoder
	encBSON encoder.Encoder

	nid []byte
}

func (t *testNodeInfo) SetupTest() {
	t.nid = []byte("test-network-id")

	t.encs = encoder.NewEncoders()
	t.encJSON = jsonenc.NewEncoder()
	t.encBSON = bsonenc.NewEncoder()

	_ = t.encs.AddEncoder(t.encJSON)
	_ = t.encs.AddEncoder(t.encBSON)

	_ = t.encs.TestAddHinter(HTTPConnInfoHinter)
	_ = t.encs.TestAddHinter(NilConnInfoHinter)
	_ = t.encs.TestAddHinter(NodeInfoV0Hinter)
	_ = t.encs.TestAddHinter(base.StringAddress(""))
	_ = t.encs.TestAddHinter(block.ManifestV0Hinter)
	_ = t.encs.TestAddHinter(key.BTCPrivatekeyHinter)
	_ = t.encs.TestAddHinter(key.BTCPublickeyHinter)
	_ = t.encs.TestAddHinter(node.BaseV0Hinter)
}

func (t *testNodeInfo) newConnInfo(name string, insecure bool) ConnInfo {
	u, _ := url.Parse(fmt.Sprintf("https://%s:443", name))

	return NewHTTPConnInfo(u, insecure)
}

func (t *testNodeInfo) newNode(name string) (base.Node, ConnInfo) {
	addr, err := base.NewStringAddress(name)
	t.NoError(err)

	no := node.NewBaseV0(addr, key.MustNewBTCPrivatekey().Publickey())

	return no, t.newConnInfo(name, true)
}

func (t *testNodeInfo) TestNew() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	local := node.RandomNode("n0")
	localConnInfo := t.newConnInfo("n0", true)

	n1, n1ConnInfo := t.newNode("n1")
	n2, n2ConnInfo := t.newNode("n2")

	nodes := []RemoteNode{
		NewRemoteNode(n1, n1ConnInfo),
		NewRemoteNode(n2, n2ConnInfo),
	}
	policy := map[string]interface{}{"showme": 1}

	suffrage := base.NewFixedSuffrage(local.Address(), nil)

	ni := NewNodeInfoV0(
		local,
		t.nid,
		base.StateBooting,
		blk.Manifest(),
		util.Version("0.1.1"),
		policy,
		nodes,
		suffrage,
		localConnInfo,
	)
	t.NoError(ni.IsValid(nil))

	t.Implements((*NodeInfo)(nil), ni)
	t.Equal(policy, ni.Policy())

	expectedNodes := []string{n1.Address().String(), n2.Address().String()}
	var regs []string
	for _, n := range ni.Nodes() {
		regs = append(regs, n.Address.String())
	}

	t.Equal(expectedNodes, regs)
	t.True(ni.ConnInfo().Equal(localConnInfo))
}

func (t *testNodeInfo) TestEmptyNetworkID() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	suffrage := base.NewFixedSuffrage(base.RandomStringAddress(), nil)

	ni := NewNodeInfoV0(
		node.RandomNode("n0"),
		nil,
		base.StateBooting,
		blk.Manifest(),
		util.Version("0.1.1"),
		map[string]interface{}{"showme": 1},
		nil,
		suffrage,
		t.newConnInfo("n0", true),
	)
	t.Contains(ni.IsValid(nil).Error(), "empty NetworkID")
}

func (t *testNodeInfo) TestWrongNetworkID() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	suffrage := base.NewFixedSuffrage(base.RandomStringAddress(), nil)

	ni := NewNodeInfoV0(
		node.RandomNode("n0"),
		t.nid,
		base.StateEmpty,
		blk.Manifest(),
		util.Version("0.1.1"),
		map[string]interface{}{"showme": 1},
		nil,
		suffrage,
		t.newConnInfo("n0", true),
	)
	t.Contains(ni.IsValid(nil).Error(), "invalid state")
}

func (t *testNodeInfo) TestEmptyBlock() {
	suffrage := base.NewFixedSuffrage(base.RandomStringAddress(), nil)
	ni := NewNodeInfoV0(
		node.RandomNode("n0"),
		t.nid,
		base.StateBooting,
		nil,
		util.Version("0.1.1"),
		map[string]interface{}{"showme": 1},
		nil,
		suffrage,
		t.newConnInfo("n0", true),
	)
	t.NoError(ni.IsValid(nil))
}

func (t *testNodeInfo) TestEmptyVersion() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	suffrage := base.NewFixedSuffrage(base.RandomStringAddress(), nil)
	ni := NewNodeInfoV0(
		node.RandomNode("n0"),
		t.nid,
		base.StateBooting,
		blk.Manifest(),
		"",
		map[string]interface{}{"showme": 1},
		nil,
		suffrage,
		t.newConnInfo("n0", true),
	)
	t.Contains(ni.IsValid(nil).Error(), "invalid version")
}

func (t *testNodeInfo) TestWrongVersion() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	suffrage := base.NewFixedSuffrage(base.RandomStringAddress(), nil)
	ni := NewNodeInfoV0(
		node.RandomNode("n0"),
		t.nid,
		base.StateBooting,
		blk.Manifest(),
		util.Version("wrong-version"),
		map[string]interface{}{"showme": 1},
		nil,
		suffrage,
		t.newConnInfo("n0", true),
	)
	t.Contains(ni.IsValid(nil).Error(), "invalid version")
}

func (t *testNodeInfo) TestJSON() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	n0, n0ConnInfo := t.newNode("n0")
	n1, n1ConnInfo := t.newNode("n1")

	nodes := []RemoteNode{
		NewRemoteNode(n0, n0ConnInfo),
		NewRemoteNode(n1, n1ConnInfo),
	}

	policy := map[string]interface{}{"showme": 1.1}

	suffrage := base.NewFixedSuffrage(base.RandomStringAddress(), nil)
	ni := NewNodeInfoV0(
		node.RandomNode("n0"),
		t.nid,
		base.StateBooting,
		blk.Manifest(),
		util.Version("1.2.3"),
		policy,
		nodes,
		suffrage,
		t.newConnInfo("n0", true),
	)
	ni.BaseHinter = hint.NewBaseHinter(hint.NewHint(NodeInfoType, "v0.0.9"))
	t.NoError(ni.IsValid(nil))

	b, err := jsonenc.Marshal(ni)
	t.NoError(err)

	i, err := DecodeNodeInfo(b, t.encJSON)
	t.NoError(err)
	nni, ok := i.(NodeInfoV0)
	t.True(ok)

	CompareNodeInfo(t.T(), ni, nni)
}

func (t *testNodeInfo) TestBSON() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	suffrage := base.NewFixedSuffrage(base.RandomStringAddress(), nil)
	ni := NewNodeInfoV0(
		node.RandomNode("n0"),
		t.nid,
		base.StateBooting,
		blk.Manifest(),
		util.Version("1.2.3"),
		map[string]interface{}{"showme": 1.1},
		nil,
		suffrage,
		t.newConnInfo("n0", true),
	)
	ni.BaseHinter = hint.NewBaseHinter(hint.NewHint(NodeInfoType, "v0.0.9"))
	t.NoError(ni.IsValid(nil))

	b, err := bsonenc.Marshal(ni)
	t.NoError(err)

	i, err := DecodeNodeInfo(b, t.encBSON)
	t.NoError(err)
	nni, ok := i.(NodeInfoV0)
	t.True(ok)

	CompareNodeInfo(t.T(), ni, nni)
}

func (t *testNodeInfo) TestSuffrage() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	suffrage := base.NewFixedSuffrage(base.RandomStringAddress(), nil)
	ni := NewNodeInfoV0(
		node.RandomNode("n0"),
		t.nid,
		base.StateBooting,
		blk.Manifest(),
		util.Version("1.2.3"),
		map[string]interface{}{"showme": 1.1},
		nil,
		suffrage,
		t.newConnInfo("n0", true),
	)
	t.NoError(ni.IsValid(nil))

	_, found := ni.Policy()["suffrage"]
	t.True(found)

	var a, b map[string]interface{}
	t.NoError(jsonenc.Unmarshal([]byte(ni.Policy()["suffrage"].(string)), &a))
	t.NoError(jsonenc.Unmarshal([]byte(suffrage.Verbose()), &b))

	t.Equal(b, a)
}

func TestNodeInfo(t *testing.T) {
	suite.Run(t, new(testNodeInfo))
}
