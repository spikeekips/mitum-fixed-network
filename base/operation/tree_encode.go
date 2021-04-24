package operation

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/tree"
)

func (no *FixedTreeNode) unpack(enc encoder.Encoder, base tree.BaseFixedTreeNode, inState bool, br []byte) error {
	no.BaseFixedTreeNode = base
	no.inState = inState

	if i, err := DecodeReasonError(enc, br); err != nil {
		return err
	} else {
		no.reason = i
	}

	return nil
}
