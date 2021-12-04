package key

import (
	"fmt"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

var (
	InvalidKeyError                  = util.NewError("invalid key")
	SignatureVerificationFailedError = util.NewError("signature verification failed")
)

type Key interface {
	fmt.Stringer
	hint.Hinter
	isvalid.IsValider
	Equal(Key) bool
	Bytes() []byte
	Raw() string // NOTE Raw() will be used for hashing. It only contains raw
	// key string without hint.
}

type Privatekey interface {
	Key
	Publickey() Publickey
	Sign([]byte) (Signature, error)
}

type Publickey interface {
	Key
	Verify([]byte, Signature) error
}

type BaseKey struct {
	hint.BaseHinter
	rawFunc func() string
}

func NewBaseKey(ht hint.Hint, rawFunc func() string) BaseKey {
	return BaseKey{BaseHinter: hint.NewBaseHinter(ht), rawFunc: rawFunc}
}

func (ky BaseKey) IsValid([]byte) error {
	return ky.BaseHinter.IsValid(nil)
}

func (ky BaseKey) Raw() string {
	return ky.rawFunc()
}

func (ky BaseKey) String() string {
	var r string
	if ky.rawFunc == nil {
		r = "<empty>"
	} else {
		r = ky.rawFunc()
	}

	return hint.NewHintedString(ky.Hint(), r).String()
}

func (ky BaseKey) Bytes() []byte {
	return []byte(ky.String())
}

func (ky BaseKey) MarshalText() ([]byte, error) {
	return []byte(ky.String()), nil
}

func (ky BaseKey) Equal(k Key) bool {
	if ky.Hint().Type() != k.Hint().Type() {
		return false
	}

	return ky.Raw() == k.Raw()
}
