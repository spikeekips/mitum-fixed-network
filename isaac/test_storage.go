// +build test

package isaac

import (
	"github.com/spikeekips/mitum/base/tree"

	"github.com/spikeekips/mitum/base/valuehash"
)

type DummyBlockStorage struct {
	block      Block
	operations *tree.AVLTree
	states     *tree.AVLTree
}

func (dst *DummyBlockStorage) Block() Block {
	return dst.block
}

func (dst *DummyBlockStorage) SetBlock(block Block) error {
	dst.block = block

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
