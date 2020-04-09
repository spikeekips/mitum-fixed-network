// +build test

package isaac

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/util/logging"
)

var log logging.Logger // nolint

func init() {
	l := zerolog.
		New(os.Stderr).
		With().
		Timestamp().
		Caller().
		Stack().
		Logger().Level(zerolog.DebugLevel)

	log = logging.NewLogger(&l, true)
}
