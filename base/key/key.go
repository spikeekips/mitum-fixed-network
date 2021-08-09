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
	Bytes() []byte // NOTE Bytes() will be used for hashing. It only contains Type
	// without version. With same type, but different version hashing will be
	// different.
	Raw() string
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
	ht      hint.Hint
	rawFunc func() string
}

func NewBaseKey(ht hint.Hint, rawFunc func() string) BaseKey {
	return BaseKey{ht: ht, rawFunc: rawFunc}
}

func (ky BaseKey) Hint() hint.Hint {
	return ky.ht
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

	return hint.NewHintedString(ky.ht, r).String()
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
