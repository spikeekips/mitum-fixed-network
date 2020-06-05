// +build test

package base

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"golang.org/x/xerrors"

	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
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

	return string(sa) == string(a.(ShortAddress))
}

func (sa ShortAddress) Bytes() []byte {
	return []byte(sa)
}

func (sa ShortAddress) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(struct {
		jsonencoder.HintedHead
		A string `json:"address"`
	}{
		HintedHead: jsonencoder.NewHintedHead(sa.Hint()),
		A:          sa.String(),
	})
}

func (sa *ShortAddress) UnpackJSON(b []byte, _ *jsonencoder.Encoder) error {
	var s struct {
		jsonencoder.HintedHead
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

func (sa ShortAddress) MarshalBSON() ([]byte, error) {
	return bsonencoder.Marshal(struct {
		HI hint.Hint `bson:"_hint"`
		A  string    `bson:"address"`
	}{
		HI: sa.Hint(),
		A:  sa.String(),
	})
}

func (sa *ShortAddress) UnmarshalBSON(b []byte) error {
	var us struct {
		A string `bson:"address"`
	}

	if err := bsonencoder.Unmarshal(b, &us); err != nil {
		return err
	}

	*sa = ShortAddress(us.A[8:])

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
