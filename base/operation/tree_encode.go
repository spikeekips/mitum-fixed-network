package operation

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/tree"
)

func (no *FixedTreeNode) unpack(enc encoder.Encoder, base tree.BaseFixedTreeNode, inState bool, br []byte) error {
	no.BaseFixedTreeNode = base
	no.inState = inState

	i, err := DecodeReasonError(br, enc)
	if err != nil {
		return err
	}
	no.reason = i

	return nil
}
