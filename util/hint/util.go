package hint

import (
	"strings"

	"github.com/spikeekips/mitum/util/isvalid"
)

func HintedString(h Hint, s string) string {
	return s + ":" + h.String()
}

func ParseHintedString(s string) (Hint, string, error) {
	i := strings.Split(s, ":")
	if len(i) < 2 {
		return Hint{}, "", isvalid.InvalidError.Errorf("invalid HintedString format found, %q", s)
	}
	h := strings.Join(i[:len(i)-1], ":")
	t := strings.Join(i[len(i)-1:], ":")

	j, err := ParseHint(t)
	if err != nil {
		return Hint{}, "", isvalid.InvalidError.Errorf("invalid Hint in HintedString found, %q", t)
	}

	return j, h, nil
}
