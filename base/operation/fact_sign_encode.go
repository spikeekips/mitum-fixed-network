package operation

import (
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
)

func (fs *BaseFactSign) unpack(
	enc encoder.Encoder,
	bSigner encoder.HintedString,
	signature key.Signature,
	signedAt time.Time,
) error {
	var signer key.Publickey
	if k, err := bSigner.Encode(enc); err != nil {
		return err
	} else if pk, ok := k.(key.Publickey); !ok {
		return xerrors.Errorf("not key.Publickey; type=%T", k)
	} else {
		signer = pk
	}

	fs.signer = signer
	fs.signature = signature
	fs.signedAt = signedAt

	return nil
}
