package keypair

import (
	"encoding/json"
	"io"

	"github.com/ethereum/go-ethereum/rlp"
)

type Signer interface {
	Sign(PrivateKey, []byte) error
}

type Kind uint

const (
	PublicKeyKind Kind = iota + 1
	PrivateKeyKind
)

func (k Kind) String() string {
	switch k {
	case PublicKeyKind:
		return "public"
	case PrivateKeyKind:
		return "private"
	}

	return ""
}

func (k Kind) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
}

type Type struct {
	id   uint
	name string
}

func NewType(id uint, name string) Type {
	return Type{id: id, name: name}
}

func (k Type) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
}

func (k Type) ID() uint {
	return k.id
}

func (k Type) Name() string {
	return k.name
}

func (k Type) Equal(b Type) bool {
	return k.id == b.id
}

func (k Type) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, k.id)
}

func (k *Type) DecodeRLP(s *rlp.Stream) error {
	var id uint
	if err := s.Decode(&id); err != nil {
		return err
	}

	k.id = id

	return nil
}

func (k Type) Empty() bool {
	return k.id < 1
}

func (k Type) String() string {
	return k.name
}
