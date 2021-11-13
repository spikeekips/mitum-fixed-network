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

type TypedString struct {
	t Type
	s string
}

func NewTypedString(t Type, s string) TypedString {
	return TypedString{t: t, s: s}
}

func ParseTypedString(s string) (TypedString, error) {
	i := strings.LastIndex(s, "~")
	if i < 1 || len(strings.TrimSpace(s[:i])) < 1 {
		return TypedString{}, errors.Errorf("invalid TypedString format found, %q; empty raw string", s)
	}

	t := Type(s[i+1:])
	if err := t.IsValid(nil); err != nil {
		return TypedString{}, fmt.Errorf("invalid Type in TypedString found, %q: %w", s[i+1:], err)
	}

	return TypedString{t: t, s: s[:i]}, nil
}

func (ts TypedString) Type() Type {
	return ts.t
}

func (ts TypedString) Body() string {
	return ts.s
}

func (ts TypedString) String() string {
	return ts.s + "~" + ts.t.String()
}

func (ts TypedString) IsValid([]byte) error {
	if err := ts.t.IsValid(nil); err != nil {
		return err
	}

	if len(ts.s) < 1 {
		return isvalid.InvalidError.Errorf("empty body string")
	}

	return nil
}
