// +build test

package localtime

import (
	"os"

	"github.com/rs/zerolog"
)

var log *zerolog.Logger // nolint

func init() {
	l := zerolog.
		New(os.Stderr).
		With().
		Timestamp().
		Caller().
		Stack().
		Logger().Level(zerolog.DebugLevel)
	log = &l
}
