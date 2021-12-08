package hint

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util/isvalid"
)

type HintedString struct {
	h Hint
	s string
}

func NewHintedString(h Hint, s string) HintedString {
	return HintedString{h: h, s: s}
}

func ParseHintedString(s string) (HintedString, error) {
	i := strings.LastIndex(s, "~")
	if i < 1 || len(strings.TrimSpace(s[:i])) < 1 {
		return HintedString{}, errors.Errorf("invalid HintedString format found, %q; empty raw string", s)
	}

	j, err := ParseHint(s[i+1:])
	if err != nil {
		return HintedString{}, fmt.Errorf("invalid Hint in HintedString found, %q: %w", s[i+1:], err)
	}
	if err := j.IsValid(nil); err != nil {
		return HintedString{}, fmt.Errorf("invalid Hint in HintedString found, %q: %w", s[i+1:], err)
	}

	return HintedString{h: j, s: s[:i]}, nil
}

func (hs HintedString) Hint() Hint {
	return hs.h
}

func (hs HintedString) Body() string {
	return hs.s
}

func (hs HintedString) String() string {
	return hs.s + "~" + hs.h.String()
}

func (hs HintedString) IsValid([]byte) error {
	if err := hs.h.IsValid(nil); err != nil {
		return err
	}

	if len(hs.s) < 1 {
		return isvalid.InvalidError.Errorf("empty body string")
	}

	return nil
}
