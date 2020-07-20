package hint

import (
	"regexp"

	"golang.org/x/xerrors"
)

var (
	REHintedStringFormat                = `(?P<s>.*)\-` + ReHintMarshalStringFormat
	REHintedStringCheck  *regexp.Regexp = regexp.MustCompile("^" + REHintedStringFormat + "$")
	REHintedString       *regexp.Regexp = regexp.MustCompile(`^(?P<s>.*)\-(?P<hint>.*)$`)
)

func HintedString(h Hint, s string) string {
	return s + "-" + h.String()
}

func ParseHintedString(s string) (Hint, string, error) {
	if !REHintedStringCheck.MatchString(s) {
		return Hint{}, "", xerrors.Errorf("unknown format of hinted string, %q", s)
	}

	var k, hs string
	if ms := REHintedString.FindStringSubmatch(s); len(ms) != 3 {
		return Hint{}, "", xerrors.Errorf("something empty of hinted string, %q", s)
	} else {
		k = ms[1]
		hs = ms[2]
	}

	var h Hint
	if i, err := NewHintFromString(hs); err != nil {
		return Hint{}, "", xerrors.Errorf("invalid hinted string, %q: %w", s, err)
	} else {
		h = i
	}

	return h, k, nil
}
