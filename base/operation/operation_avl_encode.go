package operation

import (
	"github.com/spikeekips/mitum/util/encoder"
)

func (em *OperationAVLNode) unpack(
	enc encoder.Encoder,
	key []byte,
	height int16,
	left,
	leftHash,
	right,
	rightHash,
	h,
	bOperation []byte,
) error {
	var op Operation
	if o, err := DecodeOperation(enc, bOperation); err != nil {
		return err
	} else {
		op = o
	}

	em.key = key
	em.height = height
	em.left = left
	em.leftHash = leftHash
	em.right = right
	em.rightHash = rightHash
	em.h = h
	em.op = op

	return nil
}
