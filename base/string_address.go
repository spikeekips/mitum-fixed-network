package base

import (
	"encoding/json"
	"regexp"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"

	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/logging"
)

var (
	StringAddressType = hint.MustNewType(0x01, 0x0a, "string-address")
	StringAddressHint = hint.MustHint(StringAddressType, "0.0.1")
)

var (
	reBlankAddressString *regexp.Regexp = regexp.MustCompile(`[\s][\s]*`)
	reAddressString      *regexp.Regexp = regexp.MustCompile(`^[a-zA-Z0-9][\w\-]*[a-zA-Z0-9]$`)
)

var EmptyStringAddress = StringAddress("")

type StringAddress string

func NewStringAddress(s string) (StringAddress, error) {
	sa := StringAddress(s)

	return sa, sa.IsValid(nil)
}

func NewStringAddressFromHintedString(s string) (StringAddress, error) {
	switch ht, a, err := hint.ParseHintedString(s); {
	case err != nil:
		return EmptyStringAddress, err
	case !ht.Equal(StringAddressHint):
		return EmptyStringAddress, xerrors.Errorf("not StringAddress, %v", ht)
	default:
		return NewStringAddress(a)
	}
}

func (sa StringAddress) String() string {
	return string(sa)
}

func (sa StringAddress) HintedString() string {
	return hint.HintedString(sa.Hint(), string(sa))
}

func (sa StringAddress) Hint() hint.Hint {
	return StringAddressHint
}

func (sa StringAddress) IsValid([]byte) error {
	if reBlankAddressString.Match(sa.Bytes()) {
		return isvalid.InvalidError.Errorf("address string, %q has blank", sa)
	}

	if s := strings.TrimSpace(string(sa)); len(s) < 1 {
		return isvalid.InvalidError.Errorf("empty address")
	}

	if !reAddressString.Match(sa.Bytes()) {
		return isvalid.InvalidError.Errorf("invalid address string, %q", sa)
	}

	return nil
}

func (sa StringAddress) Equal(a Address) bool {
	if sa.Hint().Type() != a.Hint().Type() {
		return false
	}

	return sa.String() == a.String()
}

func (sa StringAddress) Bytes() []byte {
	return []byte(sa.String())
}

func (sa StringAddress) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(struct {
		jsonenc.HintedHead
		A string `json:"address"`
	}{
		HintedHead: jsonenc.NewHintedHead(sa.Hint()),
		A:          sa.String(),
	})
}

func (sa *StringAddress) UnmarshalJSON(b []byte) error {
	var s struct {
		A string `json:"address"`
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	} else {
		*sa = StringAddress(s.A)
	}

	return nil
}

func (sa StringAddress) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(sa.Hint()),
		bson.M{"address": sa.String()},
	))
}

func (sa *StringAddress) UnmarshalBSON(b []byte) error {
	var s struct {
		A string `bson:"address"`
	}
	if err := bsonenc.Unmarshal(b, &s); err != nil {
		return err
	} else if len(s.A) < 1 {
		return xerrors.Errorf("not enough address")
	}

	if a, err := NewStringAddress(s.A); err != nil {
		return err
	} else {
		*sa = a
	}

	return nil
}

func (sa StringAddress) MarshalLog(key string, e logging.Emitter, _ bool) logging.Emitter {
	return e.Str(key, sa.HintedString())
}
