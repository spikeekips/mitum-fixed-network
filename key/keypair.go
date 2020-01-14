package key

import (
	"fmt"

	"github.com/spikeekips/mitum/errors"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
)

var (
	InvalidKeyError                  = errors.NewError("invalid key")
	SignatureVerificationFailedError = errors.NewError("signature verification failed")
)

type Key interface {
	fmt.Stringer
	hint.Hinter
	isvalid.IsValider
	Equal(Key) bool
}

type Publickey interface {
	Key
	Verify([]byte, Signature) error
}

type Privatekey interface {
	Key
	Publickey() Publickey
	Sign([]byte) (Signature, error)
}
