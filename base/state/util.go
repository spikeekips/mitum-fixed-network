package state

import (
	"regexp"

	"github.com/spikeekips/mitum/util/isvalid"
)

var reHasBlank = regexp.MustCompile(`\s`)

func IsValidKey(s string) error {
	if reHasBlank.Match([]byte(s)) {
		return isvalid.InvalidError.Errorf("state key should not have blank, %q", s)
	} else if len(s) < 1 {
		return isvalid.InvalidError.Errorf("empty state key")
	}

	return nil
}
