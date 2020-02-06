// +build test

package mitum

import (
	"os"

	"github.com/rs/zerolog"
)

var log zerolog.Logger // nolint

func init() {
	log = zerolog.
		New(os.Stderr).
		With().
		Timestamp().
		Caller().
		Stack().
		Logger().Level(zerolog.DebugLevel)
}
