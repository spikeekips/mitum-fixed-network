// +build test

package storage

import (
	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/tree"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type DummyBlockStorage struct {
	block      block.Block
	operations *tree.AVLTree
	states     *tree.AVLTree
	commited   bool
}

func NewDummyBlockStorage(
	blk block.Block,
	operations *tree.AVLTree,
	states *tree.AVLTree,
) *DummyBlockStorage {
	return &DummyBlockStorage{block: blk, operations: operations, states: states}
}

func (dst *DummyBlockStorage) Block() block.Block {
	return dst.block
}

func (dst *DummyBlockStorage) SetBlock(blk block.Block) error {
	dst.block = blk

	return nil
}

func (dst *DummyBlockStorage) SetOperations(tree *tree.AVLTree) error {
	dst.operations = tree

	return nil
}

func (dst *DummyBlockStorage) SetStates(tree *tree.AVLTree) error {
	dst.states = tree

	return nil
}

func (dst *DummyBlockStorage) UnstageOperationSeals([]valuehash.Hash) error {
	return nil
}

func (dst *DummyBlockStorage) Commit() error {
	dst.commited = true

	return nil
}

func (dst *DummyBlockStorage) Committed() bool {
	return dst.commited
}

type BaseTestStorage struct {
	suite.Suite
	PK      key.Privatekey
	Encs    *encoder.Encoders
	JSONEnc encoder.Encoder
	BSONEnc encoder.Encoder
}

func (t *BaseTestStorage) SetupSuite() {
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

	t.PK, _ = key.NewBTCPrivatekey()
}

func (t *BaseTestStorage) CompareManifest(a, b block.Manifest) {
	t.Equal(a.Height(), b.Height())
	t.Equal(a.Round(), b.Round())
	t.True(a.Proposal().Equal(b.Proposal()))
	t.True(a.PreviousBlock().Equal(b.PreviousBlock()))
	t.True(a.OperationsHash().Equal(b.OperationsHash()))
	t.True(a.StatesHash().Equal(b.StatesHash()))
}

func (t *BaseTestStorage) CompareBlock(a, b block.Block) {
	t.CompareManifest(a, b)
	t.Equal(a.ConsensusInfo().INITVoteproof(), b.ConsensusInfo().INITVoteproof())
	t.Equal(a.ConsensusInfo().ACCEPTVoteproof(), b.ConsensusInfo().ACCEPTVoteproof())
}
