package operation

import (
	"time"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
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
	bops []byte,
) error {
	signer, err := bSigner.Encode(enc)
	if err != nil {
		return err
	}

	hops, err := enc.DecodeSlice(bops)
	if err != nil {
		return err
	}

	sl.ops = make([]Operation, len(hops))
	for i := range hops {
		j, ok := hops[i].(Operation)
		if !ok {
			return util.WrongTypeError.Errorf("expected Operation, not %T", hops[i])
		}

		sl.ops[i] = j
	}

	sl.h = h
	sl.bodyHash = bodyHash
	sl.signer = signer
	sl.signature = signature
	sl.signedAt = signedAt

	return nil
}
