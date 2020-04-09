package tree

import (
	avlHashable "github.com/spikeekips/avl/hashable"

	"github.com/spikeekips/mitum/util/hint"
)

type Node interface {
	avlHashable.HashableNode
	hint.Hinter
	Immutable() Node
}

type NodeMutable interface {
	avlHashable.HashableMutableNode
	hint.Hinter
	Immutable() Node
}
