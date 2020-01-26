// +build test

package mitum

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/spikeekips/mitum/encoder"
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

func (sa ShortAddress) PackJSON(*encoder.JSONEncoder) (interface{}, error) {
	return struct {
		encoder.JSONPackHintedHead
		A string `json:"address"`
	}{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(sa.Hint()),
		A:                  sa.String(),
	}, nil
}

func (sa *ShortAddress) UnpackJSON(b []byte, _ *encoder.JSONEncoder) error {
	var s struct {
		A string `json:"address"`
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	} else if len(s.A) < 8 {
		return xerrors.Errorf("not enough address")
	}

	*sa = ShortAddress(s.A[8:])

	return nil
}
