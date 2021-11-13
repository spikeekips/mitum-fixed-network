package hint

import (
	"regexp"

	"github.com/spikeekips/mitum/util/isvalid"
)

var (
	reTypeAllowedChars           = regexp.MustCompile(`^[a-z0-9][a-z0-9\-_\+]*[a-z0-9]$`)
	minTypeLength, MaxTypeLength = 2, 100
)

type Type string // revive:disable-line:redefines-builtin-id

func (t Type) IsValid([]byte) error {
	switch n := len(t); {
	case n < minTypeLength:
		return isvalid.InvalidError.Errorf("empty Type")
	case n > MaxTypeLength:
		return isvalid.InvalidError.Errorf("Type too long")
	}

	if !reTypeAllowedChars.Match([]byte(t)) {
		return isvalid.InvalidError.Errorf("invalid char found in Type")
	}

	return nil
}

func (t Type) Bytes() []byte {
	return []byte(t)
}

func (t Type) String() string {
	return string(t)
}
