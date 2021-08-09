// +build test

package logging

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

var (
	TestLogging    *Logging
	TestNilLogging *Logging
)

func init() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	TestLogging = Setup(os.Stderr, zerolog.DebugLevel, "", false)
	TestNilLogging = NewLogging(nil).SetLogger(zerolog.Nop())
}
