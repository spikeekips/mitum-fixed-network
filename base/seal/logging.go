package seal

import (
	"github.com/spikeekips/mitum/util/logging"
)

func LogEventWithSeal(sl Seal, e logging.Emitter, verbose bool) logging.Emitter {
	var event logging.Emitter
	if lm, ok := sl.(logging.LogHintedMarshaler); ok {
		event = e.HintedVerbose("seal", lm, verbose)
	} else {
		event = e.
			Dict("seal", logging.Dict().
				Str("hint", sl.Hint().String()).
				Hinted("hash", sl.Hash()).(*logging.Event),
			)
	}

	return event
}
