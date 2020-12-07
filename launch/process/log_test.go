package process

import (
	"os"
	"time"

	"github.com/rs/zerolog"

	"github.com/spikeekips/mitum/util/logging"
)

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
