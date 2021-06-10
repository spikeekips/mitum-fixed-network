package operation

import (
	"time"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (sl *BaseSeal) unpack(
	enc encoder.Encoder,
	h,
	bodyHash valuehash.Hash,
	bSigner key.PublickeyDecoder,
	signature key.Signature,
	signedAt time.Time,
	operations [][]byte,
) error {
	signer, err := bSigner.Encode(enc)
	if err != nil {
		return err
	}

	sl.ops = make([]Operation, len(operations))
	for i := range operations {
		op, err := DecodeOperation(enc, operations[i])
		if err != nil {
			return err
		}
		sl.ops[i] = op
	}

	sl.h = h
	sl.bodyHash = bodyHash
	sl.signer = signer
	sl.signature = signature
	sl.signedAt = signedAt

	return nil
}
