package seal

import (
	"github.com/rs/zerolog"
)

func LogEventSeal(sl Seal, key string, e *zerolog.Event, isTrace bool) *zerolog.Event {
	if sl == nil {
		return e.Object(key, nil)
	}

	if isTrace {
		return e.Interface(key, sl)
	}

	if lm, ok := sl.(zerolog.LogObjectMarshaler); ok {
		return e.Object(key, lm)
	}

	return e.Dict(key, zerolog.Dict().
		Stringer("hint", sl.Hint()).
		Stringer("hash", sl.Hash()),
	)
}
