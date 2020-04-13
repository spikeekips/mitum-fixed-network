// +build test

package storage

import (
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/tree"
	"github.com/spikeekips/mitum/base/valuehash"
)

type DummyBlockStorage struct {
	block      block.Block
	operations *tree.AVLTree
	states     *tree.AVLTree
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
	return nil
}
