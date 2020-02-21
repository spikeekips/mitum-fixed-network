// +build test

package isaac

import (
	"github.com/spikeekips/avl"
)

type DummyBlockStorage struct {
	block      Block
	operations *avl.Tree
	states     *avl.Tree
}

func (dst *DummyBlockStorage) Block() Block {
	return dst.block
}

func (dst *DummyBlockStorage) SetOperations(tree *avl.Tree) error {
	dst.operations = tree

	return nil
}

func (dst *DummyBlockStorage) SetStates(tree *avl.Tree) error {
	dst.states = tree

	return nil
}

func (dst *DummyBlockStorage) Commit() error {
	return nil
}
