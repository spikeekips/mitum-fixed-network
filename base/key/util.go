package key

import (
	"regexp"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util/hint"
)

var (
	reKeyStringFormat                = `(?P<key>.*)\-` + hint.ReHintMarshalStringFormat
	reKeyStringCheck  *regexp.Regexp = regexp.MustCompile("^" + reKeyStringFormat + "$")
	reKeyString       *regexp.Regexp = regexp.MustCompile(`^(?P<key>.*)\-(?P<hint>.*)$`)
)

func toString(h hint.Hint, s string) string {
	return s + "-" + h.String()
}

func ParseString(s string) (hint.Hint, string, error) {
	if !reKeyStringCheck.MatchString(s) {
		return hint.Hint{}, "", xerrors.Errorf("unknown format of key string, %q", s)
	}

	var k, hs string
	if ms := reKeyString.FindStringSubmatch(s); len(ms) != 3 {
		return hint.Hint{}, "", xerrors.Errorf("something empty of key, %q", s)
	} else {
		k = ms[1]
		hs = ms[2]
	}

	var h hint.Hint
	if i, err := hint.NewHintFromString(hs); err != nil {
		return hint.Hint{}, "", xerrors.Errorf("invalid hint string of key, %q: %w", s, err)
	} else {
		h = i
	}

	return h, k, nil
}

func MustNewBTCPrivatekey() Privatekey {
	k, err := NewBTCPrivatekey()
	if err != nil {
		panic(err)
	}

	return k
}

func MustNewEtherPrivatekey() Privatekey {
	k, err := NewEtherPrivatekey()
	if err != nil {
		panic(err)
	}

	return k
}

func MustNewStellarPrivatekey() Privatekey {
	k, err := NewStellarPrivatekey()
	if err != nil {
		panic(err)
	}

	return k
}
