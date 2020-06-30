package valuehash

import (
	"github.com/spikeekips/mitum/util/logging"
)

func marshalLog(h Hash, key string, e logging.Emitter, verbose bool) logging.Emitter {
	if !verbose {
		return e.Str(key, h.String())
	}

	return e.Dict(key, logging.Dict().
		Str("hash", h.String()).
		Hinted("hint", h.Hint()),
	)
}
