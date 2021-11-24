package base

import (
	"time"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
)

func (fs *BaseFactSign) unpack(
	enc encoder.Encoder,
	bSigner key.PublickeyDecoder,
	signature key.Signature,
	signedAt time.Time,
) error {
	signer, err := bSigner.Encode(enc)
	if err != nil {
		return err
	}

	fs.signer = signer
	fs.signature = signature
	fs.signedAt = signedAt

	return nil
}
