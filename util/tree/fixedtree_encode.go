package tree

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util/encoder"
)

func (tr *FixedTree) unpack(enc encoder.Encoder, b []byte) error {
	hinters, err := enc.DecodeSlice(b)
	if err != nil {
		return err
	}

	tr.nodes = make([]FixedTreeNode, len(hinters))

	for i := range hinters {
		j, ok := hinters[i].(FixedTreeNode)
		if !ok {
			return errors.Errorf("not FixedTreeNode, %T", j)
		}
		tr.nodes[i] = j
	}

	return nil
}
