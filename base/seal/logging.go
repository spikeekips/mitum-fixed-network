package seal

import (
	"github.com/spikeekips/mitum/util/logging"
)

func LoggerWithSeal(sl Seal, e logging.Emitter, verbose bool) logging.Emitter {
	var event logging.Emitter
	if lm, ok := sl.(logging.LogHintedMarshaler); ok {
		event = e.HintedVerbose("seal", lm, verbose)
	} else {
		event = e.
			Dict("seal", logging.Dict().
				Hinted("hint", sl.Hint()).
				Hinted("hash", sl.Hash()).(*logging.Event),
			)
	}

	return event
}
