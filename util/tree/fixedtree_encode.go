package tree

import (
	"github.com/spikeekips/mitum/util/encoder"
)

func (ft *FixedTree) unpack(_ encoder.Encoder, nodes [][]byte) error {
	if t, err := NewFixedTree(nodes, nil); err != nil {
		return err
	} else {
		*ft = t
	}

	return nil
}
