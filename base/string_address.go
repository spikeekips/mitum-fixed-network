package base

import (
	"regexp"
	"strings"

	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/logging"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"golang.org/x/xerrors"
)

var (
	StringAddressType = hint.Type("sa")
	StringAddressHint = hint.NewHint(StringAddressType, "v0.0.1")
)

var (
	reBlankAddressString = regexp.MustCompile(`[\s][\s]*`)
	reAddressString      = regexp.MustCompile(`^[a-zA-Z0-9][\w\-]*[a-zA-Z0-9]$`)
)

var EmptyStringAddress = StringAddress("")

type StringAddress string

func NewStringAddress(s string) (StringAddress, error) {
	sa := StringAddress(s)

	return sa, sa.IsValid(nil)
}

func NewStringAddressFromHintedString(s string) (StringAddress, error) {
	switch hs, err := hint.ParseHintedString(s); {
	case err != nil:
		return EmptyStringAddress, err
	case !hs.Hint().Equal(StringAddressHint):
		return EmptyStringAddress, xerrors.Errorf("not StringAddress, %v", hs.Hint())
	default:
		return NewStringAddress(hs.Body())
	}
}

func (sa StringAddress) Raw() string {
	return string(sa)
}

func (sa StringAddress) String() string {
	return hint.NewHintedString(StringAddressHint, string(sa)).String()
}

func (StringAddress) Hint() hint.Hint {
	return StringAddressHint
}

func (sa StringAddress) IsValid([]byte) error {
	if reBlankAddressString.Match([]byte(sa)) {
		return isvalid.InvalidError.Errorf("address string, %q has blank", sa)
	}

	if s := strings.TrimSpace(string(sa)); len(s) < 1 {
		return isvalid.InvalidError.Errorf("empty address")
	}

	if !reAddressString.Match([]byte(sa)) {
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

func (sa StringAddress) MarshalText() ([]byte, error) {
	return []byte(sa.String()), nil
}

func (sa *StringAddress) UnmarshalText(b []byte) error {
	a, err := NewStringAddress(string(b))
	if err != nil {
		return err
	}

	*sa = a

	return nil
}

func (sa StringAddress) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bsontype.String, bsoncore.AppendString(nil, sa.String()), nil
}

func (sa StringAddress) MarshalLog(key string, e logging.Emitter, _ bool) logging.Emitter {
	return e.Str(key, sa.String())
}
