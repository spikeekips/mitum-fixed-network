package basicstates

import (
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base/seal"
)

func LogSeal(sl seal.Seal) *zerolog.Event {
	if sl == nil {
		return nil
	}

	if lm, ok := sl.(zerolog.LogObjectMarshaler); ok {
		return zerolog.Dict().EmbedObject(lm)
	}

	return zerolog.Dict().
		Stringer("hint", sl.Hint()).
		Stringer("hash", sl.Hash())
}
