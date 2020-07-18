package operation

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (oi *OperationInfoV0) unpack(_ encoder.Encoder, operation, seal valuehash.Hash) error {
	oi.oh = operation
	oi.sh = seal

	return nil
}
