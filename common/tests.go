// +build test

package common

import (
	"os"
	"runtime/debug"

	"github.com/rs/zerolog"
)

var zlog zerolog.Logger = zerolog.Nop()

func init() {
	zlog = zerolog.
		New(os.Stderr).
		With().
		Timestamp().
		Caller().
		Logger().
		Level(zerolog.DebugLevel)
}

func DebugPanic() {
	if r := recover(); r != nil {
		debug.PrintStack()
		panic(r)
	}
}
