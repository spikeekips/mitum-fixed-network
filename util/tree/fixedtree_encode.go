package tree

import (
	"github.com/spikeekips/mitum/util/encoder"
	"golang.org/x/xerrors"
)

func (tr *FixedTree) unpack(enc encoder.Encoder, bs [][]byte) error {
	tr.nodes = make([]FixedTreeNode, len(bs))

	for i := range bs {
		if j, err := enc.DecodeByHint(bs[i]); err != nil {
			return err
		} else if k, ok := j.(FixedTreeNode); !ok {
			return xerrors.Errorf("not FixedTreeNode, %T", j)
		} else {
			tr.nodes[i] = k
		}
	}

	return nil
}
