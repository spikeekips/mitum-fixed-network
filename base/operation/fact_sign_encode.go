package operation

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
	var signer key.Publickey
	if k, err := bSigner.Encode(enc); err != nil {
		return err
	} else {
		signer = k
	}

	fs.signer = signer
	fs.signature = signature
	fs.signedAt = signedAt

	return nil
}
