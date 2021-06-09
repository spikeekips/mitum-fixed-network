package hint

import (
	"strings"

	"github.com/spikeekips/mitum/util/isvalid"
)

func HintedString(h Hint, s string) string {
	return s + ":" + h.String()
}

func ParseHintedString(s string) (Hint, string, error) {
	var h, t string
	if i := strings.Split(s, ":"); len(i) < 2 {
		return Hint{}, "", isvalid.InvalidError.Errorf("invalid HintedString format found, %q", s)
	} else {
		h = strings.Join(i[:len(i)-1], ":")
		t = strings.Join(i[len(i)-1:], ":")
	}

	if i, err := ParseHint(t); err != nil {
		return Hint{}, "", isvalid.InvalidError.Errorf("invalid Hint in HintedString found, %q", t)
	} else {
		return i, h, nil
	}
}
