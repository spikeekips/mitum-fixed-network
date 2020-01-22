// +build test

package mitum

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/spikeekips/mitum/hint"
	"golang.org/x/xerrors"
)

type ShortAddress string

func NewShortAddress(s string) ShortAddress {
	return ShortAddress(s)
}

func RandomShortAddress() ShortAddress {
	b := make([]byte, 10)
	_, _ = rand.Read(b)

	return ShortAddress(hex.EncodeToString(b))
}

func (sa ShortAddress) String() string {
	return fmt.Sprintf("address:%s", string(sa))
}

func (sa ShortAddress) Hint() hint.Hint {
	h, err := hint.NewHint(
		hint.Type([2]byte{0xff, 0x40}),
		"0.1",
	)
	if err != nil {
		panic(err)
	}

	return h
}

func (sa ShortAddress) IsValid([]byte) error {
	if len(sa) < 1 {
		return xerrors.Errorf("empty address")
	}

	return nil
}

func (sa ShortAddress) Equal(a Address) bool {
	if sa.Hint().Type() != a.Hint().Type() {
		return false
	}

	return sa == a.(ShortAddress)
}

func (sa ShortAddress) Bytes() []byte {
	return []byte(sa)
}
