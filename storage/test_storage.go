// +build test

package storage

import (
	"context"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/tree"
	"github.com/spikeekips/mitum/util/valuehash"
)

type DummyDatabaseSession struct {
	block      block.Block
	operations tree.FixedTree
	states     tree.FixedTree
	commited   bool
}

func NewDummyDatabaseSession(
	blk block.Block,
	operations tree.FixedTree,
	states tree.FixedTree,
) *DummyDatabaseSession {
	return &DummyDatabaseSession{block: blk, operations: operations, states: states}
}

func (dst *DummyDatabaseSession) Block() block.Block {
	return dst.block
}

func (dst *DummyDatabaseSession) SetBlock(blk block.Block) error {
	dst.block = blk

	return nil
}

func (dst *DummyDatabaseSession) SetOperations(tree tree.FixedTree) error {
	dst.operations = tree

	return nil
}

func (dst *DummyDatabaseSession) SetStates(tree tree.FixedTree) error {
	dst.states = tree

	return nil
}

func (dst *DummyDatabaseSession) UnstageOperationSeals([]valuehash.Hash) error {
	return nil
}

func (dst *DummyDatabaseSession) Commit(context.Context) error {
	dst.commited = true

	return nil
}

func (dst *DummyDatabaseSession) Cancel() error {
	dst.commited = false

	return nil
}

func (dst *DummyDatabaseSession) Close() error {
	return dst.Cancel()
}

func (dst *DummyDatabaseSession) Committed() bool {
	return dst.commited
}

func (dst *DummyDatabaseSession) States() map[string]interface{} {
	return nil
}

type BaseTestDatabase struct {
	suite.Suite
	PK      key.Privatekey
	Encs    *encoder.Encoders
	JSONEnc encoder.Encoder
	BSONEnc encoder.Encoder
}

func (t *BaseTestDatabase) SetupSuite() {
	t.Encs = encoder.NewEncoders()
	t.JSONEnc = jsonenc.NewEncoder()
	t.BSONEnc = bsonenc.NewEncoder()

	t.NoError(t.Encs.AddEncoder(t.JSONEnc))
	t.NoError(t.Encs.AddEncoder(t.BSONEnc))

	_ = t.Encs.AddHinter(base.StringAddress(""))
	_ = t.Encs.AddHinter(base.BaseNodeV0{})
	_ = t.Encs.AddHinter(block.SuffrageInfoV0{})
	_ = t.Encs.AddHinter(key.BTCPublickeyHinter)
	_ = t.Encs.AddHinter(block.BlockV0{})
	_ = t.Encs.AddHinter(block.ManifestV0{})
	_ = t.Encs.AddHinter(block.ConsensusInfoV0{})
	_ = t.Encs.AddHinter(valuehash.SHA256{})
	_ = t.Encs.AddHinter(base.VoteproofV0{})
	_ = t.Encs.AddHinter(seal.DummySeal{})
	_ = t.Encs.AddHinter(operation.BaseSeal{})
	_ = t.Encs.AddHinter(operation.BaseFactSign{})
	_ = t.Encs.AddHinter(operation.KVOperation{})
	_ = t.Encs.AddHinter(operation.KVOperationFact{})
	_ = t.Encs.AddHinter(tree.FixedTree{})
	_ = t.Encs.AddHinter(block.BaseBlockDataMap{})

	t.PK, _ = key.NewBTCPrivatekey()
}

func (t *BaseTestDatabase) CompareManifest(a, b block.Manifest) {
	t.Equal(a.Height(), b.Height())
	t.Equal(a.Round(), b.Round())
	t.True(a.Proposal().Equal(b.Proposal()))
	t.True(a.PreviousBlock().Equal(b.PreviousBlock()))
	t.True(a.OperationsHash().Equal(b.OperationsHash()))
	t.True(a.StatesHash().Equal(b.StatesHash()))
	t.True(localtime.Equal(a.ConfirmedAt(), b.ConfirmedAt()))
}

func (t *BaseTestDatabase) CompareBlock(a, b block.Block) {
	t.CompareManifest(a, b)
	t.Equal(a.ConsensusInfo().INITVoteproof(), b.ConsensusInfo().INITVoteproof())
	t.Equal(a.ConsensusInfo().ACCEPTVoteproof(), b.ConsensusInfo().ACCEPTVoteproof())
}

func (t *BaseTestDatabase) NewBlockDataMap(height base.Height, blk valuehash.Hash, isLocal bool) block.BaseBlockDataMap {
	var scheme string = "file://"
	if !isLocal {
		scheme = "http://none-local.org"
	}

	u := func() string {
		return scheme + "/" + util.UUID().String()
	}

	bd := block.NewBaseBlockDataMap(block.TestBlockDataWriterHint, height)
	bd = bd.SetBlock(blk)

	var item block.BaseBlockDataMapItem
	item = block.NewBaseBlockDataMapItem(block.BlockDataManifest, valuehash.RandomSHA256().String(), u())
	bd, _ = bd.SetItem(item)
	item = block.NewBaseBlockDataMapItem(block.BlockDataOperations, valuehash.RandomSHA256().String(), u())
	bd, _ = bd.SetItem(item)
	item = block.NewBaseBlockDataMapItem(block.BlockDataOperationsTree, valuehash.RandomSHA256().String(), u())
	bd, _ = bd.SetItem(item)
	item = block.NewBaseBlockDataMapItem(block.BlockDataStates, valuehash.RandomSHA256().String(), u())
	bd, _ = bd.SetItem(item)
	item = block.NewBaseBlockDataMapItem(block.BlockDataStatesTree, valuehash.RandomSHA256().String(), u())
	bd, _ = bd.SetItem(item)
	item = block.NewBaseBlockDataMapItem(block.BlockDataINITVoteproof, valuehash.RandomSHA256().String(), u())
	bd, _ = bd.SetItem(item)
	item = block.NewBaseBlockDataMapItem(block.BlockDataACCEPTVoteproof, valuehash.RandomSHA256().String(), u())
	bd, _ = bd.SetItem(item)
	item = block.NewBaseBlockDataMapItem(block.BlockDataSuffrageInfo, valuehash.RandomSHA256().String(), u())
	bd, _ = bd.SetItem(item)
	item = block.NewBaseBlockDataMapItem(block.BlockDataProposal, valuehash.RandomSHA256().String(), u())
	bd, _ = bd.SetItem(item)

	i, err := bd.UpdateHash()
	t.NoError(err)
	bd = i

	t.NoError(bd.IsValid(nil))

	return bd
}
