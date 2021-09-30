package seal

import (
	"time"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (sl *BaseSeal) unpack(
	enc encoder.Encoder,
	ht hint.Hint,
	h,
	bodyHash valuehash.Hash,
	bSigner key.PublickeyDecoder,
	signature key.Signature,
	signedAt time.Time,
) error {
	signer, err := bSigner.Encode(enc)
	if err != nil {
		return err
	}

	sl.ht = ht
	sl.h = h
	sl.bodyHash = bodyHash
	sl.signer = signer
	sl.signature = signature
	sl.signedAt = signedAt

	return nil
}
