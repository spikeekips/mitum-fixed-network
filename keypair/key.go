package keypair

import (
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/spikeekips/mitum/common"
)

type Key interface {
	rlp.Encoder
	common.IsValid
	Type() common.DataType
	Kind() Kind
	Equal(Key) bool
	NativePublicKey() []byte
	String() string
}

type PublicKey interface {
	Key
	Verify([]byte, Signature) error
}

type PrivateKey interface {
	Key
	Sign([]byte) (Signature, error)
	PublicKey() PublicKey
	NativePrivateKey() []byte
}
