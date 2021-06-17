// +build test

package network

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
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

	_ = t.encs.TestAddHinter(key.BTCPrivatekeyHinter)
	_ = t.encs.TestAddHinter(key.BTCPublickeyHinter)
	_ = t.encs.TestAddHinter(base.BaseNodeV0{})
	_ = t.encs.TestAddHinter(base.StringAddress(""))
	_ = t.encs.TestAddHinter(block.ManifestV0{})
	_ = t.encs.TestAddHinter(NodeInfoV0{})
}

func (t *testNodeInfo) TestNew() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	local := base.RandomNode("n0")

	na1, err := base.NewStringAddress("n1")
	t.NoError(err)
	n1 := base.NewBaseNodeV0(na1, key.MustNewBTCPrivatekey().Publickey(), "quic://na1")

	na2, err := base.NewStringAddress("n2")
	t.NoError(err)
	n2 := base.NewBaseNodeV0(na2, key.MustNewBTCPrivatekey().Publickey(), "quic://na2")

	nodes := []base.Node{n1, n2}
	policy := map[string]interface{}{"showme": 1}

	suffrage := base.NewFixedSuffrage(local.Address(), nil)

	ni := NewNodeInfoV0(
		local,
		t.nid,
		base.StateBooting,
		blk.Manifest(),
		util.Version("0.1.1"),
		"quic://local",
		policy,
		nodes,
		suffrage,
	)
	t.NoError(ni.IsValid(nil))

	t.Implements((*NodeInfo)(nil), ni)
	t.Equal(policy, ni.Policy())

	expectedNodes := []string{n1.Address().String(), n2.Address().String(), local.Address().String()}
	var regs []string
	for _, n := range ni.Nodes() {
		regs = append(regs, n.Address().String())
	}

	t.Equal(expectedNodes, regs)
}

func (t *testNodeInfo) TestEmptyNetworkID() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	suffrage := base.NewFixedSuffrage(base.RandomStringAddress(), nil)

	ni := NewNodeInfoV0(
		base.RandomNode("n0"),
		nil,
		base.StateBooting,
		blk.Manifest(),
		util.Version("0.1.1"),
		"quic://local",
		map[string]interface{}{"showme": 1},
		nil,
		suffrage,
	)
	t.Contains(ni.IsValid(nil).Error(), "empty NetworkID")
}

func (t *testNodeInfo) TestWrongNetworkID() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	suffrage := base.NewFixedSuffrage(base.RandomStringAddress(), nil)

	ni := NewNodeInfoV0(
		base.RandomNode("n0"),
		t.nid,
		base.StateUnknown,
		blk.Manifest(),
		util.Version("0.1.1"),
		"quic://local",
		map[string]interface{}{"showme": 1},
		nil,
		suffrage,
	)
	t.Contains(ni.IsValid(nil).Error(), "invalid state")
}

func (t *testNodeInfo) TestEmptyBlock() {
	suffrage := base.NewFixedSuffrage(base.RandomStringAddress(), nil)
	ni := NewNodeInfoV0(
		base.RandomNode("n0"),
		t.nid,
		base.StateBooting,
		nil,
		util.Version("0.1.1"),
		"quic://local",
		map[string]interface{}{"showme": 1},
		nil,
		suffrage,
	)
	t.NoError(ni.IsValid(nil))
}

func (t *testNodeInfo) TestEmptyVersion() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	suffrage := base.NewFixedSuffrage(base.RandomStringAddress(), nil)
	ni := NewNodeInfoV0(
		base.RandomNode("n0"),
		t.nid,
		base.StateBooting,
		blk.Manifest(),
		"",
		"quic://local",
		map[string]interface{}{"showme": 1},
		nil,
		suffrage,
	)
	t.Contains(ni.IsValid(nil).Error(), "invalid version")
}

func (t *testNodeInfo) TestWrongVersion() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	suffrage := base.NewFixedSuffrage(base.RandomStringAddress(), nil)
	ni := NewNodeInfoV0(
		base.RandomNode("n0"),
		t.nid,
		base.StateBooting,
		blk.Manifest(),
		util.Version("wrong-version"),
		"quic://local",
		map[string]interface{}{"showme": 1},
		nil,
		suffrage,
	)
	t.Contains(ni.IsValid(nil).Error(), "invalid version")
}

func (t *testNodeInfo) TestJSON() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	na0, err := base.NewStringAddress("n0")
	t.NoError(err)
	n0 := base.NewBaseNodeV0(na0, key.MustNewBTCPrivatekey().Publickey(), "quic://na0")

	na1, err := base.NewStringAddress("n1")
	t.NoError(err)
	n1 := base.NewBaseNodeV0(na1, key.MustNewBTCPrivatekey().Publickey(), "quic://na1")

	nodes := []base.Node{n0, n1}
	policy := map[string]interface{}{"showme": 1.1}

	suffrage := base.NewFixedSuffrage(base.RandomStringAddress(), nil)
	ni := NewNodeInfoV0(
		base.RandomNode("n0"),
		t.nid,
		base.StateBooting,
		blk.Manifest(),
		util.Version("1.2.3"),
		"quic://local",
		policy,
		nodes,
		suffrage,
	)
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
		base.RandomNode("n0"),
		t.nid,
		base.StateBooting,
		blk.Manifest(),
		util.Version("1.2.3"),
		"quic://local",
		map[string]interface{}{"showme": 1.1},
		nil,
		suffrage,
	)
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
		base.RandomNode("n0"),
		t.nid,
		base.StateBooting,
		blk.Manifest(),
		util.Version("1.2.3"),
		"quic://local",
		map[string]interface{}{"showme": 1.1},
		nil,
		suffrage,
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
