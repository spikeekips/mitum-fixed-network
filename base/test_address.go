// +build test

package base

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
)

var testAddressHint = hint.MustHintWithType(hint.Type{0xff, 0x40}, "0.1", "test-address")

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
	return testAddressHint
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

func (sa ShortAddress) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(struct {
		encoder.JSONPackHintedHead
		A string `json:"address"`
	}{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(sa.Hint()),
		A:                  sa.String(),
	})
}

func (sa *ShortAddress) UnpackJSON(b []byte, _ *encoder.JSONEncoder) error {
	var s struct {
		encoder.JSONPackHintedHead
		A string `json:"address"`
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	} else if err := sa.Hint().IsCompatible(s.H); err != nil {
		return err
	} else if len(s.A) < 8 {
		return xerrors.Errorf("not enough address")
	}

	*sa = ShortAddress(s.A[8:])

	return nil
}

func (sa ShortAddress) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	if !verbose {
		return e.Str(key, sa.String())
	}

	return e.Dict(key, logging.Dict().
		Str("address", sa.String()).
		HintedVerbose("hint", sa.Hint(), true),
	)
}
