package operation

import (
	"time"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/encoder"
)

func (sl *Seal) unpack(
	enc encoder.Encoder,
	bHash,
	bBodyHash,
	bSigner []byte,
	signature key.Signature,
	signedAt time.Time,
	operations [][]byte,
) error {
	var err error
	var h, bodyHash valuehash.Hash
	if h, err = valuehash.Decode(enc, bHash); err != nil {
		return err
	}
	if bodyHash, err = valuehash.Decode(enc, bBodyHash); err != nil {
		return err
	}

	var signer key.Publickey
	if signer, err = key.DecodePublickey(enc, bSigner); err != nil {
		return err
	}

	var ops []Operation
	for _, r := range operations {
		if op, err := DecodeOperation(enc, r); err != nil {
			return err
		} else {
			ops = append(ops, op)
		}
	}

	sl.h = h
	sl.bodyHash = bodyHash
	sl.signer = signer
	sl.signature = signature
	sl.signedAt = signedAt
	sl.ops = ops

	return nil
}
