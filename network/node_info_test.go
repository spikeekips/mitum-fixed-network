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

	_ = t.encs.AddHinter(key.BTCPrivatekey{})
	_ = t.encs.AddHinter(key.BTCPublickey{})
	_ = t.encs.AddHinter(valuehash.SHA256{})
	_ = t.encs.AddHinter(base.BaseNodeV0{})
	_ = t.encs.AddHinter(base.ShortAddress(""))
	_ = t.encs.AddHinter(block.ManifestV0{})
	_ = t.encs.AddHinter(NodeInfoV0{})
}

func (t *testNodeInfo) TestNew() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	ni := NewNodeInfoV0(
		base.RandomNode("n0"),
		t.nid,
		base.StateBooting,
		blk.Manifest(),
		util.Version("0.1.1"),
		"quic://local",
		nil,
	)
	t.NoError(ni.IsValid(nil))

	t.Implements((*NodeInfo)(nil), ni)
}

func (t *testNodeInfo) TestEmptyNetworkID() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	ni := NewNodeInfoV0(
		base.RandomNode("n0"),
		nil,
		base.StateBooting,
		blk.Manifest(),
		util.Version("0.1.1"),
		"quic://local",
		nil,
	)
	t.Contains(ni.IsValid(nil).Error(), "empty NetworkID")
}

func (t *testNodeInfo) TestWrongNetworkID() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	ni := NewNodeInfoV0(
		base.RandomNode("n0"),
		t.nid,
		base.StateUnknown,
		blk.Manifest(),
		util.Version("0.1.1"),
		"quic://local",
		nil,
	)
	t.Contains(ni.IsValid(nil).Error(), "invalid state")
}

func (t *testNodeInfo) TestEmptyBlock() {
	ni := NewNodeInfoV0(
		base.RandomNode("n0"),
		t.nid,
		base.StateBooting,
		nil,
		util.Version("0.1.1"),
		"quic://local",
		nil,
	)
	t.NoError(ni.IsValid(nil))
}

func (t *testNodeInfo) TestEmptyVersion() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	ni := NewNodeInfoV0(
		base.RandomNode("n0"),
		t.nid,
		base.StateBooting,
		blk.Manifest(),
		"",
		"quic://local",
		nil,
	)
	t.Contains(ni.IsValid(nil).Error(), "invalid version")
}

func (t *testNodeInfo) TestWrongVersion() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	ni := NewNodeInfoV0(
		base.RandomNode("n0"),
		t.nid,
		base.StateBooting,
		blk.Manifest(),
		util.Version("wrong-version"),
		"quic://local",
		nil,
	)
	t.Contains(ni.IsValid(nil).Error(), "invalid version")
}

func (t *testNodeInfo) TestJSON() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	ni := NewNodeInfoV0(
		base.RandomNode("n0"),
		t.nid,
		base.StateBooting,
		blk.Manifest(),
		util.Version("1.2.3"),
		"quic://local",
		nil,
	)
	t.NoError(ni.IsValid(nil))

	b, err := jsonenc.Marshal(ni)
	t.NoError(err)

	i, err := DecodeNodeInfo(t.encJSON, b)
	t.NoError(err)
	nni, ok := i.(NodeInfoV0)
	t.True(ok)

	CompareNodeInfo(t.T(), ni, nni)
}

func (t *testNodeInfo) TestBSON() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	ni := NewNodeInfoV0(
		base.RandomNode("n0"),
		t.nid,
		base.StateBooting,
		blk.Manifest(),
		util.Version("1.2.3"),
		"quic://local",
		nil,
	)
	t.NoError(ni.IsValid(nil))

	b, err := bsonenc.Marshal(ni)
	t.NoError(err)

	i, err := DecodeNodeInfo(t.encBSON, b)
	t.NoError(err)
	nni, ok := i.(NodeInfoV0)
	t.True(ok)

	CompareNodeInfo(t.T(), ni, nni)
}

func TestNodeInfo(t *testing.T) {
	suite.Run(t, new(testNodeInfo))
}
