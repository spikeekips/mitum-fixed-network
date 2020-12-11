// +build test

package quicnetwork

import (
	"os"
	"time"

	"github.com/rs/zerolog"

	"github.com/spikeekips/mitum/util/logging"
)

//lint:file-ignore U1000 debugging inside test
var log logging.Logger

func init() {
	zerolog.TimeFieldFormat = time.RFC3339Nano

	l := zerolog.
		New(os.Stderr).
		With().
		Timestamp().
		Caller().
		Stack().
		Logger().Level(zerolog.DebugLevel)

	log = logging.NewLogger(&l, true)
}
