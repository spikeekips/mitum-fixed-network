package tree

import (
	"github.com/spikeekips/mitum/util/encoder"
	"golang.org/x/xerrors"
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
			return xerrors.Errorf("not FixedTreeNode, %T", j)
		}
		tr.nodes[i] = j
	}

	return nil
}
