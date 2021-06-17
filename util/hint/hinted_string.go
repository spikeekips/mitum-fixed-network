package hint

import (
	"strings"

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
	i := strings.Split(s, ":")
	if len(i) < 2 {
		return HintedString{}, isvalid.InvalidError.Errorf("invalid HintedString format found, %q", s)
	}
	h := strings.Join(i[:len(i)-1], ":")
	t := strings.Join(i[len(i)-1:], ":")

	j, err := ParseHint(t)
	if err != nil {
		return HintedString{}, isvalid.InvalidError.Errorf("invalid Hint in HintedString found, %q", t)
	}

	return HintedString{h: j, s: h}, nil
}

func (hs HintedString) Hint() Hint {
	return hs.h
}

func (hs HintedString) Body() string {
	return hs.s
}

func (hs HintedString) String() string {
	return hs.s + ":" + hs.h.String()
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
