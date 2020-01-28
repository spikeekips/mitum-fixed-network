package util

import (
	"regexp"

	"github.com/sanity-io/litter"
)

func init() {
	litter.Config.HidePrivateFields = false
	litter.Config.FieldExclusions = regexp.MustCompile(`^(loc|Curve)$`)
	litter.Config.Compact = false
	litter.Config.Separator = "  "
}

func Dump(i interface{}) {
	litter.Dump(i)
}
