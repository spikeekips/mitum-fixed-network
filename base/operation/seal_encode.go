package operation

import (
	"time"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

func (sl *BaseSeal) unpack(
	enc encoder.Encoder,
	h,
	bodyHash valuehash.Hash,
	bSigner encoder.HintedString,
	signature key.Signature,
	signedAt time.Time,
	operations [][]byte,
) error {
	var signer key.Publickey
	if k, err := bSigner.Encode(enc); err != nil {
		return err
	} else if pk, ok := k.(key.Publickey); !ok {
		return xerrors.Errorf("not key.Publickey; type=%T", k)
	} else {
		signer = pk
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
